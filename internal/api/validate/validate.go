package validate

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"reflect"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

var validate *validator.Validate

func init() {
	validate = validator.New(validator.WithRequiredStructEnabled())
}

func ValidateAndParseJSON(c *gin.Context, obj any) error {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return fmt.Errorf("failed to read request body: %w", err)
	}
	defer c.Request.Body.Close()
	if len(body) == 0 {
		return fmt.Errorf("request body is required")
	}

	if err := json.Unmarshal(body, obj); err != nil {
		return fmt.Errorf("invalid JSON format: %w", err)
	}
	if err := validate.Struct(obj); err != nil {
		return translateValidationError(err)
	}

	return nil
}

func ValidateAndParseQuery(c *gin.Context, obj any) error {
	queryParams := c.Request.URL.Query()

	if err := setDefaultValues(obj); err != nil {
		return err
	}

	if err := setStructFromQuery(obj, queryParams); err != nil {
		return err
	}

	if err := validate.Struct(obj); err != nil {
		return translateValidationError(err)
	}
	return nil
}

func setDefaultValues(obj any) error {
	val := reflect.ValueOf(obj).Elem()
	typ := val.Type()
	for i := range val.NumField() {
		field := val.Field(i)
		structField := typ.Field(i)
		if !field.CanSet() {
			continue
		}
		defaultValue := structField.Tag.Get("default")
		if defaultValue == "" {
			continue
		}
		if !field.IsZero() {
			continue
		}
		if err := setFieldValue(field, structField.Type, defaultValue); err != nil {
			return fmt.Errorf("invalid default value for %s: %w", structField.Name, err)
		}
	}
	return nil
}

func setStructFromQuery(obj any, queryParams url.Values) error {
	val := reflect.ValueOf(obj).Elem()
	typ := val.Type()

	for i := range val.NumField() {
		field := val.Field(i)
		structField := typ.Field(i)

		if !field.CanSet() {
			continue
		}

		paramName := structField.Tag.Get("query")
		if paramName == "" {
			continue
		}

		paramValue := queryParams.Get(paramName)
		if paramValue == "" {
			continue
		}

		if err := setFieldValue(field, structField.Type, paramValue); err != nil {
			return fmt.Errorf("invalid %s: %w", paramName, err)
		}
	}

	return nil
}

func setFieldValue(field reflect.Value, fieldType reflect.Type, value string) error {
	switch fieldType.Kind() {
	case reflect.String:
		field.SetString(value)

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		intVal, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return fmt.Errorf("must be integer")
		}
		field.SetInt(intVal)

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		uintVal, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return fmt.Errorf("must be positive integer")
		}
		field.SetUint(uintVal)

	case reflect.Float32, reflect.Float64:
		floatVal, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("must be number")
		}
		field.SetFloat(floatVal)

	case reflect.Bool:
		boolVal, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("must be true or false")
		}
		field.SetBool(boolVal)

	default:
		return fmt.Errorf("unsupported type: %s", fieldType.Kind())
	}

	return nil
}

func translateValidationError(err error) error {
	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		var errorMessages []string
		for _, e := range validationErrors {
			errorMessages = append(errorMessages, fmt.Sprintf("%s %s", e.Field(), getValidationMessage(e)))
		}
		return fmt.Errorf("%s", strings.Join(errorMessages, "; "))
	}
	return err
}

func getValidationMessage(e validator.FieldError) string {
	switch e.Tag() {
	case "required":
		return "is required"
	case "min":
		if e.Kind() == reflect.String {
			return fmt.Sprintf("must be at least %s characters", e.Param())
		}
		return fmt.Sprintf("must be at least %s", e.Param())
	case "max":
		if e.Kind() == reflect.String {
			return fmt.Sprintf("must be at most %s characters", e.Param())
		}
		return fmt.Sprintf("must be at most %s", e.Param())
	case "email":
		return "must be a valid email"
	case "numeric":
		return "must be numeric"
	case "alphanum":
		return "must contain only letters and numbers"
	case "oneof":
		return fmt.Sprintf("must be one of %s", strings.ReplaceAll(e.Param(), " ", ", "))
	default:
		return fmt.Sprintf("failed %s validation", e.Tag())
	}
}
