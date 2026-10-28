package http

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/ml8s/liki-web/internal/httputil"
	"github.com/ml8s/liki-web/internal/metrics"
)

// appFlowEventRequest is a cookieless, PII-free Consultation funnel signal.
// Every dimension is a closed vocabulary: IDs and consultation content never
// enter this endpoint, and validators prevent unbounded metric cardinality.
type appFlowEventRequest struct {
	Event    string `json:"event"`
	Domain   string `json:"domain,omitempty"`
	Source   string `json:"source,omitempty"`
	Scenario string `json:"scenario,omitempty"`
	Status   string `json:"status,omitempty"`
	Failure  string `json:"failure,omitempty"`
}

type appFlowEventContract struct {
	domain   bool
	source   map[string]bool
	scenario map[string]bool
	status   map[string]bool
	failure  bool
}

var appFlowDomains = map[string]bool{
	"natal":      true,
	"divination": true,
	"fengshui":   true,
	"naming":     true,
}

var appFlowScenarios = map[string]bool{
	"natal":      true,
	"liunian":    true,
	"hepan":      true,
	"huangli":    true,
	"divination": true,
	"fengshui":   true,
	"naming":     true,
}

var appFlowFailures = map[string]bool{
	"offline":     true,
	"unreachable": true,
	"permission":  true,
	"server":      true,
	"unknown":     true,
}

var appFlowEventContracts = map[string]appFlowEventContract{
	"consultation_entry": {
		domain:   true,
		scenario: appFlowScenarios,
		source: map[string]bool{
			"direct":          true,
			"legacy_redirect": true,
			"marketing":       true,
			"payment":         true,
		},
		status: map[string]bool{
			"empty":      true,
			"restorable": true,
		},
	},
	"consultation_domain_select": {domain: true},
	"consultation_person_select": {domain: true},
	"consultation_restore": {
		domain: true,
		source: map[string]bool{
			"latest":  true,
			"history": true,
			"payment": true,
		},
		status: map[string]bool{
			"restored": true,
			"fresh":    true,
		},
	},
	"consultation_submit": {
		domain: true,
		source: map[string]bool{
			"composer":     true,
			"focused_form": true,
			"casting":      true,
		},
		status: map[string]bool{
			"new":      true,
			"existing": true,
		},
	},
	"consultation_bridge": {
		domain:   true,
		scenario: appFlowScenarios,
		status: map[string]bool{
			"ok":              true,
			"cross_domain":    true,
			"pending_backend": true,
			"invalid":         true,
		},
	},
	"consultation_failure": {domain: true, failure: true},
	"consultation_recovery": {
		domain: true,
		source: map[string]bool{
			"workspace_retry": true,
		},
	},
}

func valueAllowed(value string, allowed map[string]bool) bool {
	return allowed[value]
}

func (r appFlowEventRequest) Validate() error {
	contract, ok := appFlowEventContracts[r.Event]
	if !ok {
		return errors.New("unknown app flow event")
	}
	if contract.domain && r.Domain == "" {
		return errors.New("missing app flow domain")
	}
	if !contract.domain && r.Domain != "" {
		return errors.New("unexpected app flow domain")
	}
	if r.Domain != "" && !valueAllowed(r.Domain, appFlowDomains) {
		return errors.New("invalid app flow domain")
	}
	if r.Source != "" && !valueAllowed(r.Source, contract.source) {
		return errors.New("invalid app flow source")
	}
	if (contract.source != nil) != (r.Source != "") {
		return errors.New("missing app flow source")
	}
	if r.Scenario != "" && !valueAllowed(r.Scenario, contract.scenario) {
		return errors.New("invalid app flow scenario")
	}
	if r.Status != "" && !valueAllowed(r.Status, contract.status) {
		return errors.New("invalid app flow status")
	}
	if (contract.status != nil) != (r.Status != "") {
		return errors.New("missing app flow status")
	}
	if (contract.failure && r.Failure == "") || (!contract.failure && r.Failure != "") ||
		(r.Failure != "" && !valueAllowed(r.Failure, appFlowFailures)) {
		return errors.New("invalid app flow failure presence")
	}
	return nil
}

func handleAppFlowEvent() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req, ok := httputil.DecodeStrict[appFlowEventRequest](w, r)
		if !ok {
			return
		}
		metrics.Default.Inc(metrics.MetricAppFlowEvent, map[string]string{
			"event":    req.Event,
			"domain":   req.Domain,
			"source":   req.Source,
			"scenario": req.Scenario,
			"status":   req.Status,
			"failure":  req.Failure,
		})
		slog.InfoContext(r.Context(), "app_flow_event",
			"event", req.Event,
			"domain", req.Domain,
			"source", req.Source,
			"scenario", req.Scenario,
			"status", req.Status,
			"failure", req.Failure,
		)
		w.WriteHeader(http.StatusNoContent)
	}
}
