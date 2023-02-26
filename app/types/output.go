package types

type Output interface {
	Write(event *Event)
}

type Outputs []Output
