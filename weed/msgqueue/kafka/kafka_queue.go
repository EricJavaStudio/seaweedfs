package kafka

import (
	"github.com/Shopify/sarama"
	"github.com/chrislusf/seaweedfs/weed/glog"
	"github.com/chrislusf/seaweedfs/weed/msgqueue"
	"github.com/chrislusf/seaweedfs/weed/util"
	"github.com/golang/protobuf/proto"
)

func init() {
	msgqueue.MessageQueues = append(msgqueue.MessageQueues, &KafkaQueue{})
}

type KafkaQueue struct {
	topic    string
	producer sarama.AsyncProducer
}

func (k *KafkaQueue) GetName() string {
	return "kafka"
}

func (k *KafkaQueue) Initialize(configuration util.Configuration) (err error) {
	glog.V(0).Infof("filer.msgqueue.kafka.hosts: %v\n", configuration.GetStringSlice("hosts"))
	glog.V(0).Infof("filer.msgqueue.kafka.topic: %v\n", configuration.GetString("topic"))
	return k.initialize(
		configuration.GetStringSlice("hosts"),
		configuration.GetString("topic"),
	)
}

func (k *KafkaQueue) initialize(hosts []string, topic string) (err error) {
	config := sarama.NewConfig()
	config.Producer.RequiredAcks = sarama.WaitForLocal
	config.Producer.Partitioner = sarama.NewHashPartitioner
	config.Producer.Return.Successes = true
	config.Producer.Return.Errors = true
	k.producer, err = sarama.NewAsyncProducer(hosts, config)
	k.topic = topic
	go k.handleSuccess()
	go k.handleError()
	return nil
}

func (k *KafkaQueue) SendMessage(key string, message proto.Message) (err error) {
	bytes, err := proto.Marshal(message)
	if err != nil {
		return
	}

	msg := &sarama.ProducerMessage{
		Topic: k.topic,
		Key:   sarama.StringEncoder(key),
		Value: sarama.ByteEncoder(bytes),
	}

	k.producer.Input() <- msg

	return nil
}

func (k *KafkaQueue) handleSuccess() {
	for {
		pm := <-k.producer.Successes()
		if pm != nil {
			glog.V(3).Infof("producer message success, partition:%d offset:%d key:%v", pm.Partition, pm.Offset, pm.Key)
		}
	}
}

func (k *KafkaQueue) handleError() {
	for {
		err := <-k.producer.Errors()
		if err != nil {
			glog.Errorf("producer message error, partition:%d offset:%d key:%v valus:%s error(%v) topic:%s", err.Msg.Partition, err.Msg.Offset, err.Msg.Key, err.Msg.Value, err.Err, k.topic)
		}
	}
}
