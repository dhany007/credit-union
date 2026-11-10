export type Staff = {
    id: number;
    name: string;
    role: "teller" | "supervisor";
    branch_id: number;
    branch_name: string;
};
export type Branch = {
    id: number;
    name: string;
};
export type Page<T> = {
    items: T[];
    next_cursor: string;
};
export type Application = {
    loan_interest_policy_id: number | null;
 approved_interest_rate_bps: number | null;
    id: number;
    application_number: string;
    member_name: string;
    account_number: string;
    amount: number;
    tenor_months: number;
    purpose: string;
    risk: string;
    status: string;
    receiving_branch: string;
    responsible_branch: string;
    decision_reason: string | null;
    approved_amount: number | null;
    approved_tenor_months: number | null;
    approval_revision: number;
    pricing_complete: boolean;
    linked_loan_id: number | null;
};
export type Account = {
 disbursement_transaction_id: string; disbursing_branch_id: number; created_by: number;
    id: number;
    application_id: number;
    loan_number: string;
    member_name: string;
    account_number: string;
    principal_amount: number;
    actual_principal_outstanding: number;
    status: string;
    responsible_branch: string;
    annual_rate_bps: number;
    tenor_months: number;
    current_schedule_version_id: number;
    disbursed_at: string;
};
export type Installment = {
    id?: number;
    installment_number: number;
    due_date: string;
    opening_principal?: number;
    principal_due: number;
    interest_due: number;
    total_due: number;
    closing_principal?: number;
    principal_paid?: number;
    interest_paid?: number;
    principal_unpaid?: number;
    interest_unpaid?: number;
};
export type Preview = {
    amount: number;
    tenor_months: number;
    annual_rate_bps: number;
    policy_id: number;
    regular_installment_amount: number;
    total_interest: number;
    total_payable: number;
    installments: Installment[];
    base_rate_fallback?: boolean;
    payout_date?: string;
    approval_revision?: number;
    available_cash?: number;
    payout_eligible?: boolean;
};
export type Allocation = {
    installment_number?: number;
    schedule_id: number;
    principal_amount: number;
    interest_amount: number;
};
export type PaymentPreview = {
    amount: number;
    expected_payment_revision: number;
    current_schedule_version_id: number;
    outstanding_payable: number;
    principal_before: number;
    principal_after: number;
    allocations: Allocation[];
};
export type Version = {
    id: number;
    version_number: number;
    version_type: string;
    is_current_version: boolean;
    effective_date: string;
    reason: string;
    approved_by: number;
    approved_at: string;
    interest_rate_bps: number;
    tenor_months: number;
};
export type Payment = {
    id: number;
    repayment_number: string;
    amount: number;
    paid_at: string;
    transaction_id: string;
    branch: string;
    actor: string;
    note: string;
    allocations: Allocation[];
};
export type Command = {
    path: string;
    payload: string;
    key: string;
};
export type Receipt = {
    label?: string;
    actor?: string;
    branch?: string;
    reference?: string;
    amount?: number;
    loan_id?: number;
    loan_number?: string;
    application_id?: number;
    disbursed_at?: string;
    paid_at?: string;
    repayment_number?: string;
    principal_after?: number;
    allocations?: Allocation[];
    id?: number;
};
export function jakartaToday(now = new Date()) { return new Date(now.getTime() + 7 * 3600000).toISOString().slice(0, 10); }
export function payoutBody(p: Preview, verification: string) { return { expected_approval_revision: p.approval_revision, expected_policy_id: p.policy_id, expected_rate_bps: p.annual_rate_bps, expected_principal: p.amount, expected_tenor_months: p.tenor_months, expected_payout_date: p.payout_date, recipient_verified: true, agreement_signed: true, conditions_verified: true, verification_note: verification, note: "" }; }
export function repaymentBody(p: PaymentPreview, note: string) { return { amount: p.amount, expected_payment_revision: p.expected_payment_revision, expected_schedule_version_id: p.current_schedule_version_id, note }; }
export function loanListReference(item: Application | Account): string { return "application_number" in item ? item.application_number : item.loan_number; }
