package advocate

import (
	"encoding/json"
	"testing"
)

func TestInitializationIdentifiesStorageMode(t *testing.T) {
	for _, local := range []bool{false, true} {
		s := &Server{Log: inaccessibleTestLog{}, Local: local}
		response, reply := s.handle(t.Context(), &rpcRequest{
			ID: json.RawMessage(`1`), Method: "initialize", Params: json.RawMessage(`{}`),
		})
		if !reply || response.Error != nil {
			t.Fatalf("local=%t initialization: %+v", local, response)
		}
		result := response.Result.(map[string]any)
		want := advocateInstructions
		if local {
			want = localAdvocateInstructions
		}
		if result["instructions"] != want {
			t.Fatalf("local=%t instructions = %q", local, result["instructions"])
		}
	}
}
