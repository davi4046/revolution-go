package component

// CreateGenerator creates a new generator component in the current working directory.
func CreateGenerator(name string) error {

	if err := createComponent(name, "generator"); err != nil {
		return err
	}

	return nil
}
