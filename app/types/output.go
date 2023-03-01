package types

type Output interface {
	Write(event *Event)
}

type Outputs []Output

func (o Outputs) Write(event *Event) {
	for _, output := range o {
		output.Write(event)
	}
}
