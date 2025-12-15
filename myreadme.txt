brew install kubebuilder

cd ~/code/go/cyk

mkdir k8s-opreator

cd k8s-opreator

go mod init k8s-opreator

kubebuilder init --domain cyk.io

kubebuilder create api --group myservice --version v1 --kind Apiservice

在api/apiservice_types.go中的spec和status中添加自己想要的字段

make manifests