# log-flexvolume
flexvolume for log aggregator。挂载指定目录到宿主机，方便收集容器日志

1、 mkdir /usr/libexec/kubernetes/kubelet-plugins/volume/exec/applog.io~log-flexvolume

2、cp log-flexvolume /usr/libexec/kubernetes/kubelet-plugins/volume/exec/applog.io~log-flexvolume/log-flexvolume

3、define volume
        "volumes": [
          {
            "name": "app-log",
            "flexVolume": {
              "driver": "applog.io/log-flexvolume",
              "fsType": "xfs",
              "options": {
                "format": "nginx"
              }
            }
          }
        ]

4、mount volume
            "volumeMounts": [
              {
                "name": "app-log",
                "mountPath": "/usr/local/kong/logs"
              }
            ]
