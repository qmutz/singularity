module github.com/sylabs/singularity

go 1.11

require (
	github.com/alexflint/go-filemutex v0.0.0-20171028004239-d358565f3c3f // indirect
	github.com/containerd/cgroups v0.0.0-20200824123100-0b889c03f102
	github.com/containernetworking/cni v0.8.0
	github.com/containernetworking/plugins v0.8.6
	github.com/containers/image v0.0.0-20180612162315-2e4f799f5eba
	github.com/containers/storage v1.28.1 // indirect
	github.com/cpuguy83/go-md2man v1.0.10 // indirect
	github.com/docker/docker v0.0.0-20180522102801-da99009bbb11 // indirect
	github.com/docker/docker-credential-helpers v0.6.0 // indirect
	github.com/docker/go-connections v0.3.0 // indirect
	github.com/docker/libtrust v0.0.0-20160708172513-aabc10ec26b7 // indirect
	github.com/dsnet/compress v0.0.1 // indirect
	github.com/globalsign/mgo v0.0.0-20180615134936-113d3961e731
	github.com/godbus/dbus v4.1.0+incompatible // indirect
	github.com/gorilla/websocket v1.4.2
	github.com/gotestyourself/gotestyourself v2.2.0+incompatible // indirect
	github.com/kelseyhightower/envconfig v1.3.0
	github.com/kr/pty v1.1.5
	github.com/kubernetes-sigs/cri-o v0.0.0-20180917213123-8afc34092907
	github.com/magiconair/properties v1.8.0
	github.com/mattn/go-colorable v0.1.1 // indirect
	github.com/mtrmac/gpgme v0.0.0-20170102180018-b2432428689c // indirect
	github.com/opencontainers/image-spec v1.0.1
	github.com/opencontainers/image-tools v0.0.0-00010101000000-000000000000
	github.com/opencontainers/runtime-spec v1.0.3-0.20200929063507-e6143ca7d51d
	github.com/opencontainers/runtime-tools v0.6.0
	github.com/opencontainers/selinux v1.8.0
	github.com/pelletier/go-toml v1.2.0
	github.com/pkg/errors v0.9.1
	github.com/satori/go.uuid v1.2.0
	github.com/seccomp/libseccomp-golang v0.9.1
	github.com/spf13/cobra v0.0.3
	github.com/spf13/pflag v1.0.5
	github.com/sylabs/json-resp v0.5.0
	github.com/sylabs/scs-key-client v0.2.0
	github.com/sylabs/sif v1.0.3
	go4.org v0.0.0-20180417224846-9599cf28b011 // indirect
	golang.org/x/crypto v0.0.0-20201002170205-7f63de1d35b0
	golang.org/x/sys v0.0.0-20210124154548-22da62e12c0c
	gopkg.in/cheggaaa/pb.v1 v1.0.25
	gopkg.in/yaml.v2 v2.3.0
	gotest.tools v2.3.0+incompatible // indirect
	k8s.io/client-go v11.0.0+incompatible // indirect
)

replace (
	github.com/Sirupsen/logrus => github.com/sirupsen/logrus v1.0.5
	github.com/opencontainers/image-tools => github.com/sylabs/image-tools v0.0.0-20181006203805-2814f4980568
	golang.org/x/crypto => github.com/sylabs/golang-x-crypto v0.0.0-20181006204705-4bce89e8e9a9
)
