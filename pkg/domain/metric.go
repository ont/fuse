package domain

type Metric interface {
	Save(msg Message) error
}
