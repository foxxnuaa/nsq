package main

import (
	"github.com/bitly/nsq/util"
	"sort"
)

type TopicStats struct {
	TopicName    string         `json:"topic_name"`
	Channels     []ChannelStats `json:"channels"`
	Depth        int64          `json:"depth"`
	BackendDepth int64          `json:"backend_depth"`
	MessageCount uint64         `json:"message_count"`

	E2eProcessingLatency *util.PercentileResult `json:"e2e_processing_latency"`
}

func NewTopicStats(t *Topic, channels []ChannelStats) TopicStats {
	return TopicStats{
		TopicName:    t.name,
		Channels:     channels,
		Depth:        t.Depth(),
		BackendDepth: t.backend.Depth(),
		MessageCount: t.messageCount,

		E2eProcessingLatency: t.AggregateChannelE2eProcessingLatency().PercentileResult(),
	}
}

type ChannelStats struct {
	ChannelName   string        `json:"channel_name"`
	Depth         int64         `json:"depth"`
	BackendDepth  int64         `json:"backend_depth"`
	InFlightCount int           `json:"in_flight_count"`
	DeferredCount int           `json:"deferred_count"`
	MessageCount  uint64        `json:"message_count"`
	RequeueCount  uint64        `json:"requeue_count"`
	TimeoutCount  uint64        `json:"timeout_count"`
	Clients       []ClientStats `json:"clients"`
	Paused        bool          `json:"paused"`

	E2eProcessingLatency *util.PercentileResult `json:"e2e_processing_latency"`
}

func NewChannelStats(c *Channel, clients []ClientStats) ChannelStats {
	return ChannelStats{
		ChannelName:   c.name,
		Depth:         c.Depth(),
		BackendDepth:  c.backend.Depth(),
		InFlightCount: len(c.inFlightMessages),
		DeferredCount: len(c.deferredMessages),
		MessageCount:  c.messageCount,
		RequeueCount:  c.requeueCount,
		TimeoutCount:  c.timeoutCount,
		Clients:       clients,
		Paused:        c.IsPaused(),

		E2eProcessingLatency: c.e2eProcessingLatencyStream.PercentileResult(),
	}
}

type ClientStats struct {
	Version       string `json:"version"`
	RemoteAddress string `json:"remote_address"`
	Name          string `json:"name"`
	State         int32  `json:"state"`
	ReadyCount    int64  `json:"ready_count"`
	InFlightCount int64  `json:"in_flight_count"`
	MessageCount  uint64 `json:"message_count"`
	FinishCount   uint64 `json:"finish_count"`
	RequeueCount  uint64 `json:"requeue_count"`
	ConnectTime   int64  `json:"connect_ts"`
}

type Topics []*Topic

func (t Topics) Len() int      { return len(t) }
func (t Topics) Swap(i, j int) { t[i], t[j] = t[j], t[i] }

type TopicsByName struct {
	Topics
}

func (t TopicsByName) Less(i, j int) bool { return t.Topics[i].name < t.Topics[j].name }

type Channels []*Channel

func (c Channels) Len() int      { return len(c) }
func (c Channels) Swap(i, j int) { c[i], c[j] = c[j], c[i] }

type ChannelsByName struct {
	Channels
}

func (c ChannelsByName) Less(i, j int) bool { return c.Channels[i].name < c.Channels[j].name }

func (n *NSQd) getStats() []TopicStats {
	n.RLock()
	defer n.RUnlock()

	realTopics := make([]*Topic, 0, len(n.topicMap))
	for _, t := range n.topicMap {
		realTopics = append(realTopics, t)
	}
	sort.Sort(TopicsByName{realTopics})

	topics := make([]TopicStats, 0, len(n.topicMap))
	for _, t := range realTopics {
		t.RLock()

		realChannels := make([]*Channel, 0, len(t.channelMap))
		for _, c := range t.channelMap {
			realChannels = append(realChannels, c)
		}
		sort.Sort(ChannelsByName{realChannels})

		channels := make([]ChannelStats, 0, len(t.channelMap))
		for _, c := range realChannels {
			c.RLock()
			clients := make([]ClientStats, 0, len(c.clients))
			for _, client := range c.clients {
				clients = append(clients, client.Stats())
			}
			channels = append(channels, NewChannelStats(c, clients))
			c.RUnlock()
		}

		topics = append(topics, NewTopicStats(t, channels))

		t.RUnlock()
	}

	return topics
}
