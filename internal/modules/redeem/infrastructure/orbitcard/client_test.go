package orbitcard

import "testing"

func TestParseCreateResultSubmissionOutcome(t *testing.T) {
	tests := []struct {
		name       string
		httpStatus int
		body       string
		want       string
	}{
		{"accepted", 200, `{"code":0,"data":{"task_no":"task-1"}}`, "accepted"},
		{"unknown business error", 200, `{"code":15000}`, "unknown"},
		{"unknown client error", 400, `{"code":15000}`, "unknown"},
		{"unknown error with task", 200, `{"code":15000,"data":{"task_no":"task-1"}}`, "unknown"},
		{"invalid credential", 200, `{"code":59004}`, "rejected"},
		{"expired session", 200, `{"code":40027}`, "rejected"},
		{"server error", 502, `{"code":59004}`, "unknown"},
		{"protocol error", 200, `{"code":59018}`, "unknown"},
		{"explicit rejection", 502, `{"code":59018,"data":{"request_error":{"submission_outcome":"rejected"}}}`, "rejected"},
		{"explicit unknown", 200, `{"code":59004,"data":{"request_error":{"submission_outcome":"unknown"}}}`, "unknown"},
		{"request error with task", 200, `{"code":0,"data":{"task_no":"task-1","request_error":{"submission_outcome":"unknown"}}}`, "unknown"},
		{"unrecognized outcome", 200, `{"code":59018,"data":{"request_error":{"submission_outcome":"accepted"}}}`, "unknown"},
		{"missing outcome", 200, `{"code":59004,"data":{"request_error":{}}}`, "unknown"},
		{"missing task", 200, `{"code":0,"data":{}}`, "unknown"},
		{"invalid response", 200, `<html>error</html>`, "unknown"},
		{"unauthenticated", 401, `{}`, "rejected"},
		{"forbidden", 403, `{}`, "rejected"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseCreateResult(tt.httpStatus, []byte(tt.body))
			if got.SubmissionOutcome != tt.want {
				t.Fatalf("outcome = %q, want %q", got.SubmissionOutcome, tt.want)
			}
		})
	}
}

func TestParseCreateResultPreservesUnknownOrder(t *testing.T) {
	got := parseCreateResult(200, []byte(`{"code":15000,"data":{"task_no":"task-1","client_order_no":"client-1","request_error":{"code":"relay_error","submission_outcome":"unknown"}}}`))
	if got.BusinessCode != 15000 || got.TaskNo != "task-1" || got.ClientOrderNo != "client-1" || got.ErrorCode != "relay_error" {
		t.Fatalf("lost original order diagnostics: %+v", got)
	}
}
