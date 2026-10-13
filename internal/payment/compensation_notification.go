package payment

// NotificationEntitlementCompensation identifies customer notifications for
// support-granted Chat entitlements. It shares the durable email queue with
// order receipts and uses the same retry/audit semantics.
const NotificationEntitlementCompensation NotificationKind = "entitlement_compensation"

// EntitlementCompensationKind keeps the queue topic in one place. It exists
// because composition-root code should not hard-code notification kind strings.
func EntitlementCompensationKind() NotificationKind {
	return NotificationEntitlementCompensation
}
