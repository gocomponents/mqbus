package mqbus

type Message interface {
	ConnectionString()string
	Exchange() string
	MessageType() string
	RoutingKey() string
}


