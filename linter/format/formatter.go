package format

type Formatter interface {
	Format(results []error) (string, error)
}
