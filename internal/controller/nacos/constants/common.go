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

	LabelNacosStandalone = "nacos.yunweizhan.com.cn/nacos-standalone"
	LabelNacosCluster    = "nacos.yunweizhan.com.cn/nacos-cluster"
	LabelApp             = "app"
	DefaultImage         = "docker.io/nacos/nacos-server"

	DefaultNacosLivenessPath  = "/nacos/v2/console/health/liveness"
	DefaultNacosReadinessPath = "/nacos/v2/console/health/readiness"
)

type JDBCTemplate string

const (
	DefaultMysqlJDBCTemplate JDBCTemplate = "jdbc:mysql://%s:%s/%s?characterEncoding=utf8&connectTimeout=1000&socketTimeout=3000&autoReconnect=true&useUnicode=true&useSSL=false&serverTimezone=UTC"
	DefaultDatabaseName                   = "nacos"
)

const (
	EnvDatabasePlatform = "spring.datasource.platform"
	EnvDBNum            = "db.num"
	EnvDBUser           = "db.user"
	EnvDBPassword       = "db.password"
	EnvDBUrlPrefix      = "db.url."
	EnvMode             = "MODE"
	EnvMemberList       = "MEMBER_LIST"
)

const (
	NacosModeStandalone = "standalone"
	NacosModeCluster    = "cluster"
)

const (
	NodeLabelHostName = "kubernetes.io/hostname"
)
