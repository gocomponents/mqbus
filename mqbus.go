package mqbus

import (
	"encoding/json"
	"fmt"
	"github.com/streadway/amqp"
	"strings"
	"sygit.suiyi.com.cn/go/core/util/pool"
	"sync"
	"time"
)

type amqpConnection struct {
	Conn *amqp.Connection
}

func (cp *amqpConnection) Dispose() error {
	if cp.Conn != nil {
		return cp.Conn.Close()
	}

	return nil
}

type mqBus struct {
	onceMap       sync.Map
	connectionMap sync.Map
	mu            sync.Mutex
}

var Default = &mqBus{
	onceMap:       sync.Map{},
	connectionMap: sync.Map{},
	mu:            sync.Mutex{},
}

const (
	Host        = "host="
	UserName    = "username="
	PassWord    = "password="
	VirtualHost = "virtualHost="
)

func (mqbus *mqBus) get(connectionString string) (pool.Pool, error) {
	v, ok := mqbus.connectionMap.Load(connectionString)

	if ok {
		return v.(pool.Pool), nil
	} else {
		mqbus.mu.Lock()

		once, _ := mqbus.onceMap.LoadOrStore(connectionString, &sync.Once{})

		var err error

		once.(*sync.Once).Do(func() {
			uri := connectionString //"host=10.1.4.131:5672;username=guest;password=guest"
			var connectionPool pool.Pool
			var connection *amqp.Connection
			connectionPool, err = pool.NewChannelPool(3, 10, func() (pool.Object, error) {
				url := parse2Amqp(uri)
				connection, err = amqp.Dial(url)

				if err != nil {
					return nil, err
				}

				if connection == nil {
					return nil, fmt.Errorf("connection nil")
				}

				amqp1 := &amqpConnection{Conn: connection}

				receiver := make(chan *amqp.Error)

				amqp1.Conn.NotifyClose(receiver)

				go func(v *amqpConnection) {
					for {
						select {
						case <-receiver:
							{
							Loop:
								for {
									if v != nil {
										v.Dispose()
									}
									url := parse2Amqp(uri)
									connection, err = amqp.Dial(url)

									if err == nil {
										v.Conn = connection
										receiver = make(chan *amqp.Error)
										v.Conn.NotifyClose(receiver)
										fmt.Println("重连成功", &v)
										break Loop
									} else {
										fmt.Println("重连失败,5s后重试", err)

										if connection != nil {
											connection.Close()
										}
										time.Sleep(5 * time.Second)
									}
								}

							}
						default:
							time.Sleep(5 * time.Second)
						}
					}
				}(amqp1)

				return amqp1, nil
			})

			if err == nil && connectionPool != nil {
				mqbus.connectionMap.Store(connectionString, connectionPool)
			}

		})
		mqbus.mu.Unlock()

		v, ok := mqbus.connectionMap.Load(connectionString)

		if !ok {
			return nil, fmt.Errorf("mqbus.connectionMap.Load !ok")
		}
		return v.(pool.Pool), nil
	}
}

func subIndexStr(str string, indexStr string) string {

	if str == "" || indexStr == "" {
		return ""
	}
	rs := []rune(str)

	result := string(rs[len(indexStr):len(str)])

	return result
}

func parse2Amqp(amqpConnectionString string) string {

	if amqpConnectionString == "" {
		return ""
	}

	kv := strings.Split(amqpConnectionString, ";")

	var host string
	var username string
	var password string
	var vHost string
	for _, v := range kv {
		if strings.Contains(v, Host) {
			host = subIndexStr(v, Host)
		}
		if strings.Contains(v, UserName) {
			username = subIndexStr(v, UserName)
		}
		if strings.Contains(v, PassWord) {
			password = subIndexStr(v, PassWord)
		}
		if strings.Contains(v, VirtualHost) {
			vHost = subIndexStr(v, VirtualHost)
		}
	}

	if vHost == "" {
		return "amqp://" + username + ":" + password + "@" + host
	} else {
		return "amqp://" + username + ":" + password + "@" + host + "/" + vHost
	}
}

func (mqbus *mqBus) Post(message Message) error {

	connectionPool, err := mqbus.get(message.ConnectionString())

	if err != nil {
		return err
	}

	pooledObj, err := connectionPool.Get()

	if err != nil {
		return err
	}

	defer pooledObj.Dispose()

	if err != nil {
		return fmt.Errorf("connectionPool.Get: %s", err)
	}

	object := pooledObj.(*pool.PooledObject)

	conn := object.Obj.(*amqpConnection)

	connection := conn.Conn

	channel, err := connection.Channel()

	if err != nil {
		return fmt.Errorf("Channel: %v , Error : %v", &conn, err)
	}

	defer channel.Close()

	//if true {
	//	if err := channel.Confirm(false); err != nil {
	//		return fmt.Errorf("Channel could not be put into confirm mode: %s", err)
	//	}
	//
	//	confirms := channel.NotifyPublish(make(chan amqp.Confirmation, 1))
	//
	//	defer confirmOne(confirms)
	//}

	//var headers = amqp.Table{}
	//
	//headers["x-delay"]=int32(1)

	body, err := json.Marshal(&message)

	if err = channel.Publish(
		message.Exchange(),   // publish to an exchange
		message.RoutingKey(), // routing to 0 or more queues
		false,                // mandatory
		false,                // immediate
		amqp.Publishing{
			Headers:         amqp.Table{}, //headers,
			ContentType:     "text/plain",
			Type:            message.MessageType(),
			ContentEncoding: "",
			Body:            body,
			DeliveryMode:    amqp.Transient, // 1=non-persistent, 2=persistent
			Priority:        0,              // 0-9
			// a bunch of application/implementation-specific fields
		},
	); err != nil {
		return fmt.Errorf("Exchange Publish: %s", err)
	}

	return nil
}

func confirmOne(confirms <-chan amqp.Confirmation) {

	if confirmed := <-confirms; confirmed.Ack {
		//log.Printf("confirmed delivery with delivery tag: %d", confirmed.DeliveryTag)
	} else {
		//log.Printf("failed delivery of delivery tag: %d", confirmed.DeliveryTag)
	}
}
