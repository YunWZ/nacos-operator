package constants

const (
	DefaultNacosServerHttpPort     = 8848
	DefaultNacosServerHttpPortName = "client-http"
	DefaultNacosServerGrpcPort     = 9848
	DefaultNacosServerGrpcPortName = "client-grpc"

	DefaultNacosServerRaftPort           = 7848
	DefaultNacosServerRaftPortName       = "raft"
	DefaultNacosServerPeerToPeerPort     = 9849
	DefaultNacosServerPeerToPeerPortName = "peer-to-peer"

	//DefaultNacosServerGrpcPort = 9848
	//DefaultNacosServerGrpcPortName = "client-Grpc"
	LabelNacosStandalone = "nacos.yunweizhan.com.cn/nacos-standalone"
	LabelNacosCluster    = "nacos.yunweizhan.com.cn/nacos-cluster"
	LabelApp             = "app"
	DefaultImage         = "docker.io/nacos/nacos-server"
)
