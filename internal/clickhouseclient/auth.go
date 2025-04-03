package clickhouseclient

type UserPasswordAuth struct {
	Username string
	Password string
	Database string
}

func (u *UserPasswordAuth) ValidateConfig() (bool, []string) {
	errors := make([]string, 0)
	if u.Username == "" {
		errors = append(errors, "Username must be set")
	}

	return len(errors) == 0, errors
}

type BasicAuth struct {
	Username string
	Password string
}

func (b *BasicAuth) ValidateConfig() (bool, []string) {
	errors := make([]string, 0)
	if b.Username == "" {
		errors = append(errors, "Username must be set")
	}

	return len(errors) == 0, errors
}
