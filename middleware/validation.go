// middleware/validation.go
package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strconv"
	"strings"
)

// ValidationError represents a validation error
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ValidationResponse represents the response for validation errors
type ValidationResponse struct {
	Status  string            `json:"status"`
	Message string            `json:"message"`
	Errors  []ValidationError `json:"errors"`
}

// ValidateRequest validates the request body against validation rules
func ValidateRequest(next http.HandlerFunc, validator interface{}) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Check if the request has a body
		if r.Body == nil {
			http.Error(w, "Request body is required", http.StatusBadRequest)
			return
		}

		// Read the request body
		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read request body", http.StatusBadRequest)
			return
		}

		// Close the original body
		r.Body.Close()

		// Create a new instance of the validator struct
		val := reflect.New(reflect.TypeOf(validator)).Interface()

		// Decode the request body
		err = json.Unmarshal(bodyBytes, val)
		if err != nil {
			http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
			return
		}

		// Validate the struct
		errors := validateStruct(val)
		if len(errors) > 0 {
			// Return validation errors
			response := ValidationResponse{
				Status:  "error",
				Message: "Validation failed",
				Errors:  errors,
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(response)
			return
		}

		// Reset the body for the next handler to read
		r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

		// Call the next handler
		next(w, r)
	}
}

// validateStruct validates a struct using reflection
func validateStruct(s interface{}) []ValidationError {
	var errors []ValidationError

	// Get reflect value and type
	v := reflect.ValueOf(s).Elem()
	t := v.Type()

	// Iterate over struct fields
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		value := v.Field(i)

		// Get validation tags
		tag := field.Tag.Get("validate")
		if tag == "" {
			continue
		}

		// Get JSON name if available, otherwise use field name
		jsonName := field.Tag.Get("json")
		if jsonName == "" {
			jsonName = strings.ToLower(field.Name)
		} else {
			// Handle cases like "json:"field_name,omitempty"
			parts := strings.Split(jsonName, ",")
			jsonName = parts[0]
		}

		// Validate field
		validationErrors := validateField(value, tag, jsonName)
		errors = append(errors, validationErrors...)
	}

	return errors
}

// validateField validates a field against its validation rules
func validateField(value reflect.Value, tag string, fieldName string) []ValidationError {
	var errors []ValidationError

	// Split validation tags
	validations := strings.Split(tag, ",")

	for _, validation := range validations {
		// Handle different validation types
		if validation == "required" {
			if isEmptyValue(value) {
				errors = append(errors, ValidationError{
					Field:   fieldName,
					Message: "This field is required",
				})
			}
		} else if strings.HasPrefix(validation, "min=") {
			// Handle min validation for string and numeric types
			minStr := strings.TrimPrefix(validation, "min=")
			err := validateMin(value, minStr, fieldName)
			if err != nil {
				errors = append(errors, ValidationError{
					Field:   fieldName,
					Message: err.Error(),
				})
			}
		}
		// Add more validation types as needed
	}

	return errors
}

// isEmptyValue checks if a value is empty
func isEmptyValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.String:
		return v.String() == ""
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Slice, reflect.Map:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Interface, reflect.Ptr:
		return v.IsNil()
	}
	return false
}

// validateMin validates minimum constraints
func validateMin(v reflect.Value, minStr string, fieldName string) error {
	switch v.Kind() {
	case reflect.String:
		minLen, err := strconv.Atoi(minStr)
		if err != nil {
			return fmt.Errorf("Invalid minimum value: %s", minStr)
		}
		if v.Len() < minLen {
			return fmt.Errorf("String length must be at least %s", minStr)
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		minVal, err := strconv.ParseInt(minStr, 10, 64)
		if err != nil {
			return fmt.Errorf("Invalid minimum value: %s", minStr)
		}
		if v.Int() < minVal {
			return fmt.Errorf("Value must be at least %s", minStr)
		}
	case reflect.Float32, reflect.Float64:
		minVal, err := strconv.ParseFloat(minStr, 64)
		if err != nil {
			return fmt.Errorf("Invalid minimum value: %s", minStr)
		}
		if v.Float() < minVal {
			return fmt.Errorf("Value must be at least %s", minStr)
		}
	}
	return nil
}
