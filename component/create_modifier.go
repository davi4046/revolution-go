package component

// CreateModifier creates a new modifier component in the current working directory.
func CreateModifier(name string) error {

	if err := createComponent(name, "modifier"); err != nil {
		return err
	}

	return nil
}
