package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"reflect"
	"regexp"
	"sync"
	"testing"

	"credit-union/backend/internal/platform/database"
)

type accountMember struct {
	ID            int64  `json:"id"`
	Number        string `json:"number"`
	AccountNumber string `json:"account_number"`
	Balance       int64  `json:"balance"`
	HomeBranchID  int    `json:"home_branch_id"`
}

func decodeAccount(t *testing.T, w *httptest.ResponseRecorder) accountMember {
	t.Helper()
	requireStatus(t, w, 200)
	var result accountMember
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	return result
}

func TestSavingsAccountRegistrationAndLookup(t *testing.T) {
	f := newFixture(t)
	payload := `{"nik":"3271010101950001","name":"Account Test","phone":"08123456789","address":"Synthetic address only"}`
	original := f.call("teller.a", "POST", "/api/members", "account-register-001", payload)
	registered := decodeAccount(t, original)
	if registered.AccountNumber != "1000000001" || registered.Number != "CU-000001" {
		t.Fatalf("distinct member/account identifiers missing: %+v", registered)
	}
	retried := decodeAccount(t, f.call("teller.a", "POST", "/api/members", "account-register-001", payload))
	if !reflect.DeepEqual(registered, retried) {
		t.Fatal("retry changed the account or registration result")
	}
	found := decodeAccount(t, f.call("teller.b", "GET", "/api/members/by-account/"+registered.AccountNumber, "", ""))
	if found.ID != registered.ID || found.HomeBranchID != 1 || found.Balance != 50000 {
		t.Fatalf("cross-KCP lookup resolved wrong member: %+v", found)
	}
	requireStatus(t, f.postCash("teller.b", "account-cross-kcp-001", "deposit", found.ID, 25000), 200)
	after := decodeAccount(t, f.call("teller.a", "GET", "/api/members/by-account/"+registered.AccountNumber, "", ""))
	if after.AccountNumber != registered.AccountNumber || after.Balance != 75000 {
		t.Fatal("deposit changed account identity or wrong balance")
	}
	search := f.call("teller.b", "GET", "/api/members?search="+registered.AccountNumber, "", "")
	requireStatus(t, search, 200)
	var matches []accountMember
	if err := json.Unmarshal(search.Body.Bytes(), &matches); err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 || matches[0].ID != registered.ID {
		t.Fatal("full account number is not searchable")
	}
	requireStatus(t, f.call("supervisor", "GET", "/api/members/by-account/"+registered.AccountNumber, "", ""), 200)
	requireStatus(t, f.call("", "GET", "/api/members/by-account/"+registered.AccountNumber, "", ""), 401)
	for _, invalid := range []string{"100000", "10000000010", "abcdefghij", "0000000000"} {
		requireStatus(t, f.call("teller.b", "GET", "/api/members/by-account/"+invalid, "", ""), 400)
	}
	requireStatus(t, f.call("teller.b", "GET", "/api/members/by-account/9999999998", "", ""), 404)
}

func TestSavingsAccountConcurrentAllocation(t *testing.T) {
	f := newFixture(t)
	const registrations = 6
	results := make(chan *httptest.ResponseRecorder, registrations)
	var wg sync.WaitGroup
	for i := 0; i < registrations; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			actor := "teller.a"
			if i%2 != 0 {
				actor = "teller.b"
			}
			payload := fmt.Sprintf(
				`{"nik":"327101010195%04d","name":"Concurrent Account","phone":"08123456789","address":"Synthetic address only"}`,
				i+10,
			)
			results <- f.call(actor, "POST", "/api/members", fmt.Sprintf("account-concurrent-%04d", i), payload)
		}(i)
	}
	wg.Wait()
	close(results)
	numbers := map[string]bool{}
	for result := range results {
		member := decodeAccount(t, result)
		if !regexp.MustCompile(`^[1-9][0-9]{9}$`).MatchString(member.AccountNumber) || numbers[member.AccountNumber] {
			t.Fatalf("invalid or duplicate allocated account: %q", member.AccountNumber)
		}
		numbers[member.AccountNumber] = true
	}
}

func TestSavingsAccountMigrationBackfill(t *testing.T) {
	f := newFixtureWithMigration(t, false)
	ctx := context.Background()
	_, err := f.db.Exec(ctx, `
        INSERT INTO members(nik,name,phone,address,home_branch_id,balance) VALUES
            ('3271010101950021','Legacy One','08123456789','Synthetic address',1,50000),
            ('3271010101950022','Legacy Two','08123456789','Synthetic address',2,75000);
        INSERT INTO transactions(id,member_id,branch_id,actor_id,kind,amount)
            SELECT 'legacy-'||m.id,m.id,m.home_branch_id,u.id,'deposit',m.balance
            FROM members m JOIN users u ON u.username='teller.a';
        INSERT INTO journal_lines(transaction_id,line_no,account,debit,credit)
            SELECT id,1,'cash',amount,0 FROM transactions;
        INSERT INTO journal_lines(transaction_id,line_no,account,debit,credit)
            SELECT id,2,'member_savings',0,amount FROM transactions;`)
	if err != nil {
		t.Fatal(err)
	}
	snapshot := func() string {
		t.Helper()
		var result string
		err := f.db.QueryRow(ctx, `
            SELECT jsonb_build_object(
                'members',(SELECT jsonb_agg(to_jsonb(m)-'account_number' ORDER BY id) FROM members m),
                'transactions',(SELECT jsonb_agg(to_jsonb(t)-'balance_before'-'balance_after' ORDER BY id)
                    FROM transactions t),
                'journal',(SELECT jsonb_agg(j ORDER BY transaction_id,line_no) FROM journal_lines j),
                'requests',(SELECT jsonb_agg(r ORDER BY user_id,request_key) FROM requests r)
            )::text`).Scan(&result)
		if err != nil {
			t.Fatal(err)
		}
		return result
	}
	before := snapshot()
	if err = database.Migrate(ctx, f.db); err != nil {
		t.Fatal(err)
	}
	if snapshot() != before {
		t.Fatal("migration changed existing membership or financial data")
	}
	readAccounts := func() []string {
		t.Helper()
		var accounts []string
		if err := f.db.QueryRow(ctx, `SELECT array_agg(account_number ORDER BY id) FROM members`).
			Scan(&accounts); err != nil {
			t.Fatal(err)
		}
		return accounts
	}
	accounts := readAccounts()
	if len(accounts) != 2 || accounts[0] == accounts[1] {
		t.Fatal("backfill did not allocate distinct accounts")
	}
	for _, number := range accounts {
		if !regexp.MustCompile(`^[1-9][0-9]{9}$`).MatchString(number) {
			t.Fatal("invalid backfill number")
		}
	}
	migrationErrors := make(chan error, 2)
	var wg sync.WaitGroup
	for range 2 {
		wg.Add(1)
		go func() { defer wg.Done(); migrationErrors <- database.Migrate(ctx, f.db) }()
	}
	wg.Wait()
	close(migrationErrors)
	for err := range migrationErrors {
		if err != nil {
			t.Fatal(err)
		}
	}
	if !reflect.DeepEqual(accounts, readAccounts()) || snapshot() != before {
		t.Fatal("rerun renumbered accounts or changed funds")
	}
	next := f.member(t, "3271010101950023", "teller.b")
	var number string
	if err = f.db.QueryRow(ctx, `SELECT account_number FROM members WHERE id=$1`, next).Scan(&number); err != nil {
		t.Fatal(err)
	}
	if number == accounts[0] || number == accounts[1] {
		t.Fatal("new account reused a backfilled number")
	}
	if _, err = f.db.Exec(ctx, `UPDATE members SET account_number=$1 WHERE id=$2`, accounts[0], next); err == nil {
		t.Fatal("database accepted duplicate account number")
	}
	if _, err = f.db.Exec(ctx, `UPDATE members SET account_number='123' WHERE id=$1`, next); err == nil {
		t.Fatal("database accepted malformed account number")
	}
	if _, err = f.db.Exec(ctx, `UPDATE members SET account_number=NULL WHERE id=$1`, next); err == nil {
		t.Fatal("database accepted null account number")
	}
}

func TestSavingsAccountSequenceExhaustionIsAtomic(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	if _, err := f.db.Exec(ctx, `ALTER SEQUENCE savings_account_number_seq RESTART WITH 9999999999`); err != nil {
		t.Fatal(err)
	}
	first := f.member(t, "3271010101950031", "teller.a")
	var number string
	if err := f.db.QueryRow(ctx, `SELECT account_number FROM members WHERE id=$1`, first).Scan(&number); err != nil {
		t.Fatal(err)
	}
	if number != "9999999999" {
		t.Fatal("last valid account number was not allocated")
	}
	payload := `{"nik":"3271010101950032","name":"No Account Capacity",` +
		`"phone":"08123456789","address":"Synthetic address"}`
	requireStatus(t, f.call("teller.b", "POST", "/api/members", "account-exhausted-001", payload), 500)
	var members, transactions, requests int
	var balance int64
	if err := f.db.QueryRow(ctx, `SELECT count(*),coalesce(sum(balance),0) FROM members`).
		Scan(&members, &balance); err != nil {
		t.Fatal(err)
	}
	if err := f.db.QueryRow(ctx, `SELECT count(*) FROM transactions`).Scan(&transactions); err != nil {
		t.Fatal(err)
	}
	if err := f.db.QueryRow(ctx, `SELECT count(*) FROM requests`).Scan(&requests); err != nil {
		t.Fatal(err)
	}
	if members != 1 || balance != 50000 || transactions != 1 || requests != 1 {
		t.Fatal("exhausted numbering left partial registration or funds")
	}
}

func TestSavingsAccountLookupBeyondListLimit(t *testing.T) {
	f := newFixture(t)
	target := f.member(t, "3271010101950041", "teller.a")
	ctx := context.Background()
	_, err := f.db.Exec(ctx, `
        INSERT INTO members(nik,name,phone,address,home_branch_id)
        SELECT lpad(n::text,16,'0'),'Synthetic later member','00000000','Synthetic address',1
        FROM generate_series(1,101) AS n
    `)
	if err != nil {
		t.Fatal(err)
	}
	list := f.call("teller.b", "GET", "/api/members", "", "")
	requireStatus(t, list, 200)
	var recent []accountMember
	if err = json.Unmarshal(list.Body.Bytes(), &recent); err != nil {
		t.Fatal(err)
	}
	if len(recent) != 100 {
		t.Fatal("fixture must exceed the existing recent-member limit")
	}
	for _, member := range recent {
		if member.ID == target {
			t.Fatal("old target unexpectedly in recent list")
		}
	}
	found := decodeAccount(t, f.call("teller.b", "GET", "/api/members/by-account/1000000001", "", ""))
	if found.ID != target || found.Balance != 50000 {
		t.Fatal("exact account lookup was limited by recent-member list")
	}
}
