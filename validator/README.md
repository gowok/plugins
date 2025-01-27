# `validator`
It help to do validations.

## Installation

```bash
go get github.com/gowok/plugins/validator
```

## Validating
```go
type User struct {
    Email string `validate:"required,email"`
}

user := User{}
errs := validator.ValidateStruct(user, nil)
fmt.Println(errs)
```

output:
```
// text
User.Email: Email is a required field;

// json
{"User.Email":"Email is a required field"}
```
