package model

import (
	"encoding/json"
	"net/http"
	"reflect"
	"testing"
)

func testErrorMarshalingWithData(t *testing.T) {
	tests := []struct {
		name   string
		appErr AppError
	}{
		{
			name:   "default",
			appErr: AppError{},
		},
		{
			name:   "no data",
			appErr: AppError{Status: http.StatusBadRequest, Code: MissingUsers},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			marshaled, err := json.Marshal(tt.appErr)
			if err != nil {
				t.Fatalf("failed marshaling: %s", err.Error())
			}

			var unmarshaledError AppError
			err = json.Unmarshal(marshaled, &unmarshaledError)
			if err != nil {
				t.Fatalf("failed unmarshaling: %s", err.Error())
			}

			if !reflect.DeepEqual(tt.appErr, unmarshaledError) {
				t.Errorf("expecting: %+v, got: %+v", activityTypeComparison, unmarshaledError)
			}
		})
	}
}
