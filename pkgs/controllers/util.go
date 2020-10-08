package controllers

import (
	v1 "github.com/cloudflare/origin-ca-issuer/pkgs/apis/v1"
	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/clock"
)

// IssuerHasCondition will return true if the given OriginIssuer has a condition
// matching the provided OriginIssuerCondtion. Only the Type and Status fields
// are used in the comparison, meaning this function will return `true` even if
// the Reason, Message, and LastTransitionTime fields do not match.
func IssuerHasCondition(iss v1.OriginIssuer, c v1.OriginIssuerCondition) bool {
	for _, cond := range iss.Status.Conditions {
		if c.Type == cond.Type && c.Status == cond.Status {
			return true
		}
	}

	return false
}

// SetIssuerCondition will set a condition on the given OriginIssuer.
//
// If no condition of the same type exists, the condition will be inserted with
// the LastTransitionTime set to the current time.
//
// If a condition of the same type and status already exists, the condition will
// be updated but the LastTransitionTime will no be modified.
//
// If a condition of the same type and different state already exists, the
// condition will be updated and the LastTransitionTime set to the current
// time.
func SetIssuerCondition(iss *v1.OriginIssuer, conditionType v1.ConditionType, status v1.ConditionStatus, log logr.Logger, cl clock.Clock, reason, message string) {
	now := metav1.NewTime(cl.Now())
	c := v1.OriginIssuerCondition{
		Type:               conditionType,
		Status:             status,
		Reason:             reason,
		Message:            message,
		LastTransitionTime: &now,
	}

	for i, condition := range iss.Status.Conditions {
		if condition.Type != conditionType {
			continue
		}

		if condition.Status == status {
			c.LastTransitionTime = condition.LastTransitionTime
		} else {
			log.Info("found status change for OriginIssuer; setting lastTransitionTime",
				"condition", condition.Type,
				"old_status", condition.Status,
				"new_status", c.Status,
			)
		}

		iss.Status.Conditions[i] = c

		return
	}

	iss.Status.Conditions = append(iss.Status.Conditions, c)
}
