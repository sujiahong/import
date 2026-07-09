package su_config

type Validator interface {
	Validate() error
}

func validate(out any) error {
	if v, ok := out.(Validator); ok {
		return v.Validate()
	}
	return nil
}
