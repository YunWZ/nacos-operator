package constants

const (
	DefaultNacosServerHttpPort     = 8848
	DefaultNacosServerHttpPortName = "client-http"
	DefaultNacosServerGrpcPort     = 9848
	DefaultNacosServerGrpcPortName = "client-Grpc"

	DefaultNacosServerRaftPort     = 9849
	DefaultNacosServerRaftPortName = "raft"

	//DefaultNacosServerGrpcPort = 9848
	//DefaultNacosServerGrpcPortName = "client-Grpc"
	LabelNacosStandalone = "nacos.yunweizhan.com.cn/nacos-standalone"
	LabelNacosCluster    = "nacos.yunweizhan.com.cn/nacos-cluster"
	LabelApp             = "app"
	DefaultImage         = "docker.io/verdgun/nacos"
)
