package component

import "reflect"

func getEmptyFields(s interface{}) []string {
	emptyFields := []string{}

	value := reflect.ValueOf(s)
	if value.Kind() == reflect.Ptr {
		value = value.Elem()
	}

	for i := 0; i < value.NumField(); i++ {
		fieldValue := value.Field(i)
		fieldType := value.Type().Field(i)

		// Check if the field is a string and empty
		if fieldType.Type.Kind() == reflect.String && fieldValue.String() == "" {
			emptyFields = append(emptyFields, fieldType.Name)
		}

		// Check if the field is a zero value
		if reflect.DeepEqual(fieldValue.Interface(), reflect.Zero(fieldType.Type).Interface()) {
			emptyFields = append(emptyFields, fieldType.Name)
		}
	}

	return emptyFields
}
