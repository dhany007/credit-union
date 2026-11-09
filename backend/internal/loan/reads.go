package loan

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"

	"credit-union/backend/internal/platform/fault"

	"github.com/jackc/pgx/v5"
)

const maxLoanReferenceLength = 50

const applicationDataSQL = `SELECT a.id,to_jsonb(a)||jsonb_build_object(
 'application_number','PJ-'||a.id::text,'member_name',m.name,'account_number',m.account_number,
 'receiving_branch',b.name,'responsible_branch',rb.name,
 'linked_loan_id',l.id,'loan_number',l.loan_number,
 'pricing_complete',a.approved_amount IS NOT NULL,
 'approval_ready',a.status='approved' AND a.approved_amount IS NOT NULL AND l.id IS NULL
  AND m.is_active AND m.deleted_at IS NULL AND rb.is_active AND rb.deleted_at IS NULL) data
 FROM loan_applications a JOIN members m ON m.id=a.member_id
 JOIN branches b ON b.id=a.receiving_branch_id JOIN branches rb ON rb.id=a.responsible_branch_id
 LEFT JOIN loans l ON l.application_id=a.id`
const accountDataSQL = `SELECT l.id,to_jsonb(l)||jsonb_build_object(
 'member_name',m.name,'account_number',m.account_number,'responsible_branch',b.name,
 'actual_principal_outstanding',l.principal_amount-(SELECT COALESCE(sum(a.principal_amount),0)
 FROM loan_repayment_allocations a WHERE a.loan_id=l.id),
 'current_schedule_version_id',v.id,'annual_rate_bps',v.interest_rate_bps,'policy_id',
v.loan_interest_policy_id,
 'tenor_months',v.tenor_months,'interest_method',v.interest_method) data
 FROM loans l JOIN members m ON m.id=l.member_id JOIN branches b ON b.id=l.responsible_branch_id
 JOIN loan_schedule_versions v ON v.loan_id=l.id AND v.is_current_version`

func validFilters(v url.Values, allowed ...string) error {
	for key, values := range v {
		found := false
		for _, name := range allowed {
			if name == key {
				found = true
				break
			}
		}
		if !found || len(values) != 1 {
			return fault.New(fault.Invalid, "Invalid loan filter")
		}
	}
	return nil
}
func listFilters(v url.Values, app bool) ([]any, error) {
	if err := validFilters(
		v,
		"cursor",
		"account_number",
		"status",
		"responsible_branch_id",
		"application_number",
		"loan_number",
		"payout_pending",
	); err != nil {
		return nil, err
	}
	account := v.Get("account_number")
	if account != "" {
		n, err := strconv.ParseInt(account, 10, 64)
		if err != nil || len(account) != 10 || n < 1000000000 {
			return nil, fault.New(fault.Invalid, "Invalid account number")
		}
	}
	status := v.Get("status")
	if !app && status == "" {
		status = "active"
	}
	if status == "all" {
		status = ""
	}
	if status != "" && ((app && status != "submitted" && status != "approved" && status != "rejected") ||
		(!app && status != "active" && status != "paid_off")) {
		return nil, fault.New(fault.Invalid, "Invalid loan status")
	}
	branch := int64(0)
	if raw := v.Get("responsible_branch_id"); raw != "" {
		var err error
		branch, err = parseID(raw)
		if err != nil {
			return nil, err
		}
	}
	pending := false
	if raw := v.Get("payout_pending"); raw != "" {
		if !app || (raw != "true" && raw != "false") {
			return nil, fault.New(fault.Invalid, "Invalid payout filter")
		}
		pending = raw == "true"
	}
	reference := v.Get("loan_number")
	if app {
		if reference != "" {
			return nil, fault.New(fault.Invalid, "Invalid application reference filter")
		}
		reference = v.Get("application_number")
	} else if v.Get("application_number") != "" {
		return nil, fault.New(fault.Invalid, "Invalid loan reference filter")
	}
	if len(reference) > maxLoanReferenceLength {
		return nil, fault.New(fault.Invalid, "Loan reference is too long")
	}
	return []any{account, status, branch, reference, pending}, nil
}

// ApplicationPage exposes new pagination without changing the legacy /loans array.
func (s *Service) ApplicationPage(ctx context.Context, v url.Values) (Page, error) {
	args, err := listFilters(v, true)
	if err != nil {
		return Page{}, err
	}
	sql := applicationDataSQL + ` WHERE a.is_active AND a.deleted_at IS NULL
 AND ($1::text='' OR m.account_number=$1) AND ($2::text='' OR a.status=$2)
 AND ($3::bigint=0 OR a.responsible_branch_id=$3)
 AND ($4::text='' OR 'PJ-'||a.id::text=$4)
 AND (NOT $5::boolean OR (a.status='approved' AND l.id IS NULL))`
	return s.page(ctx, sql, args, v.Get("cursor"), queryScope("applications", v), false)
}

// AccountPage includes historical members and paid-off contracts when requested.
func (s *Service) AccountPage(ctx context.Context, v url.Values) (Page, error) {
	args, err := listFilters(v, false)
	if err != nil {
		return Page{}, err
	}
	sql := accountDataSQL + ` WHERE ($1::text='' OR m.account_number=$1) AND ($2::text='' OR l.status=$2)
 AND ($3::bigint=0 OR l.responsible_branch_id=$3) AND ($4::text='' OR l.loan_number=$4) AND NOT
$5::boolean`
	return s.page(ctx, sql, args, v.Get("cursor"), queryScope("accounts", v), false)
}

// Detail preserves historical attribution without filtering inactive master joins.
func (s *Service) Detail(ctx context.Context, id int64, application bool) (json.RawMessage, error) {
	sql := accountDataSQL + " WHERE l.id=$1"
	if application {
		sql = applicationDataSQL + " WHERE a.id=$1 AND a.is_active AND a.deleted_at IS NULL"
	}
	var ignored int64
	var data json.RawMessage
	err := s.store.pool.QueryRow(ctx, sql, id).Scan(&ignored, &data)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fault.New(fault.NotFound, "Loan resource not found")
	}
	return data, err
}

// History returns scope-bound pages of schedules, versions or committed receipts.
func (s *Service) History(ctx context.Context, id int64, kind string, v url.Values) (Page, error) {
	if _, err := s.Detail(ctx, id, false); err != nil {
		return Page{}, err
	}
	allowed := []string{"cursor"}
	if kind == "installments" {
		allowed = append(allowed, "schedule_version_id")
	}
	if err := validFilters(v, allowed...); err != nil {
		return Page{}, err
	}
	args := []any{id}
	ascending := false
	var sql string
	switch kind {
	case "schedule-versions":
		sql = `SELECT v.version_number::bigint id,to_jsonb(v) data FROM loan_schedule_versions v WHERE v.loan_id=$1`
	case "installments":
		var version int64
		var err error
		if raw := v.Get("schedule_version_id"); raw != "" {
			version, err = parseID(raw)
			if err != nil {
				return Page{}, err
			}
		}
		err = s.store.pool.QueryRow(ctx, `SELECT id FROM loan_schedule_versions WHERE loan_id=$1
   AND (($2::bigint=0 AND is_current_version) OR id=$2)`, id, version).Scan(&version)
		if errors.Is(err, pgx.ErrNoRows) {
			return Page{}, fault.New(fault.NotFound, "Schedule version not found")
		}
		if err != nil {
			return Page{}, err
		}
		args = append(args, version)
		ascending = true
		sql = `SELECT s.installment_number::bigint id,to_jsonb(s)||jsonb_build_object(
   'total_due',s.principal_due+s.interest_due,'principal_paid',COALESCE(sum(a.principal_amount),0),
   'interest_paid',COALESCE(sum(a.interest_amount),0),
   'principal_unpaid',s.principal_due-COALESCE(sum(a.principal_amount),0),
   'interest_unpaid',s.interest_due-COALESCE(sum(a.interest_amount),0)) data
   FROM loan_schedules s LEFT JOIN loan_repayment_allocations a ON a.schedule_id=s.id
   WHERE s.loan_id=$1 AND s.schedule_version_id=$2 GROUP BY s.id`
	case "repayments":
		sql = `SELECT r.id,to_jsonb(r)||jsonb_build_object('branch',b.name,'actor',u.name,
   'allocations',COALESCE((SELECT jsonb_agg(to_jsonb(a) ORDER BY a.id)
    FROM loan_repayment_allocations a WHERE a.repayment_id=r.id),'[]'::jsonb)) data
   FROM loan_repayments r JOIN branches b ON b.id=r.servicing_branch_id
   LEFT JOIN users u ON u.id=r.created_by WHERE r.loan_id=$1`
	default:
		return Page{}, fmt.Errorf("unsupported loan history")
	}
	return s.page(ctx, sql, args, v.Get("cursor"), queryScope(fmt.Sprintf("loan:%d:%s", id, kind), v), ascending)
}
