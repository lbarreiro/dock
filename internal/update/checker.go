package update

type Checker interface {

Name() string

Check(image string) (Result, error)

}
