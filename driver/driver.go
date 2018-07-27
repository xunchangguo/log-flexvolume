package driver

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"

	valid "github.com/asaskevich/govalidator"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

const (
	mountCmd      = "mount"
	unmountCmd    = "umount"
	svcLogBaseDir = "/var/lib/app/log-volumes"
)

var (
	mountCmdArg = []string{"-o", "bind"}
)

type Options struct {
	Format       string `json:"format,omitempty" valid:"required"`
	PodName      string `json:"kubernetes.io/pod.name,omitempty"`
	PodNameSpace string `json:"kubernetes.io/pod.namespace,omitempty"`
	PodUid       string `json:"kubernetes.io/pod.uid,omitempty"`
}

//  kubernetes.io/fsType   kubernetes.io/pod.name   kubernetes.io/pvOrVolumeName
//  kubernetes.io/pod.namespace kubernetes.io/pod.uid kubernetes.io/readwrite
//  kubernetes.io/serviceAccount.name

type FlexVolumeDriver struct {
	Logger *logrus.Logger
}

func (f *FlexVolumeDriver) Init() InitResponse {
	if err := precreateDir(); err != nil {
		return InitResponse{
			CommonResponse: returnErrorResponse(err),
		}
	}

	if err := os.MkdirAll(svcLogBaseDir, os.ModePerm); err != nil {
		return InitResponse{
			CommonResponse: returnErrorResponse(fmt.Errorf("create log dir %s failed, %v", svcLogBaseDir, err)),
		}
	}

	return InitResponse{
		CommonResponse: CommonResponse{
			Status:  StatusSuccess,
			Message: "Success",
		},
		Capabilities: struct {
			Attach bool `json:"attach"`
		}{
			Attach: false,
		},
	}
}

func (f *FlexVolumeDriver) Mount(args []string) CommonResponse {
	var err error
	defer func(logger *logrus.Logger) {
		if err != nil {
			logger.Error(err)
		}
	}(f.Logger)
	// param check
	f.Logger.Debugf("mount args: %v", args)
	if err = checkArgsLen(args, 2); err != nil {
		return returnErrorResponse(err)
	}

	containerPath := args[0]
	opts := Options{}
	if err = json.Unmarshal([]byte(args[1]), &opts); err != nil {
		return returnErrorResponse(err)
	}

	if _, err = valid.ValidateStruct(opts); err != nil {
		return returnErrorResponse(err)
	}

	//generate config
	if err := precreateDir(); err != nil {
		return returnErrorResponse(err)
	}

	var hostDir string
	hostDir = path.Join(svcLogBaseDir, opts.PodName+"_"+opts.PodNameSpace+"_"+opts.PodUid)

	if err = os.MkdirAll(hostDir, os.ModePerm); err != nil {
		return returnErrorResponse(fmt.Errorf("create hostPath failed, %v", err))
	}

	if err = bindMount(hostDir, containerPath); err != nil {
		return returnErrorResponse(fmt.Errorf("bind mount failed, %v", err))
	}
	return CommonResponse{
		Status:  StatusSuccess,
		Message: "Success",
	}
}

func (f *FlexVolumeDriver) Unmount(args []string) CommonResponse {
	var err error
	defer func(logger *logrus.Logger) {
		if err != nil {
			logger.Error(err)
		}
	}(f.Logger)

	f.Logger.Debugf("ummount args: %v", args)
	if err = checkArgsLen(args, 1); err != nil {
		return returnErrorResponse(err)
	}

	containerPath := args[0]
	if err = unMount(containerPath); err != nil {
		return returnErrorResponse(fmt.Errorf("unmount container path %s failed, %v", containerPath, err))
	}

	return CommonResponse{
		Status:  StatusSuccess,
		Message: "Success",
	}
}

func bindMount(hostPath string, containerPath string) error {
	c := append(mountCmdArg, hostPath)
	c = append(c, containerPath)
	cmd := exec.Command(mountCmd, c...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("run bind mount command failed, hostPath: %s, containerPath: %s, error: %v, output: %s", hostPath, containerPath, err, string(output))
	}
	return nil
}

func unMount(containerPath string) error {
	cmd := exec.Command(unmountCmd, containerPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf(string(output))
	}
	return nil
}

func checkArgsLen(args cli.Args, expectedNum int) error {
	if len(args) < expectedNum {
		err := fmt.Errorf("mount: invalid args num, %v", args)
		return err
	}
	return nil
}

func returnErrorResponse(err error) CommonResponse {
	return CommonResponse{
		Status:  StatusFailure,
		Message: fmt.Sprintf("%v", err),
	}
}

func isConfigEqual(file1, file2 string) error {
	f1, err := ioutil.ReadFile(file1)
	if err != nil {
		return errors.Wrapf(err, "fail read file %s", file1)
	}

	f2, err := ioutil.ReadFile(file2)

	if err != nil {
		return errors.Wrapf(err, "fail read file %s", file2)
	}
	if bytes.Equal(f1, f2) {
		return nil
	}
	return fmt.Errorf("file not equal")
}

func copyFileContent(fromPath, toPath string) error {
	from, err := os.Open(fromPath)
	if err != nil {
		return errors.Wrapf(err, "fail to open tmp config file")
	}
	defer from.Close()

	to, err := os.OpenFile(toPath, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return errors.Wrap(err, "fail to open current config file")
	}
	defer to.Close()

	_, err = io.Copy(to, from)
	if err != nil {
		return errors.Wrap(err, "fail to copy config file")
	}
	if err = to.Sync(); err != nil {
		return errors.Wrap(err, "fail to sync config file")
	}
	return nil
}

func precreateDir() error {
	if err := os.MkdirAll(svcLogBaseDir, os.ModePerm); err != nil {
		return fmt.Errorf("create log dir %s failed, %v", svcLogBaseDir, err)
	}
	return nil
}

func removeFiles(files []string) error {
	for _, v := range files {
		if err := os.Remove(v); err != nil {
			return errors.Wrapf(err, "remove file %s failed", v)
		}
	}
	return nil
}

func isContain(obj string, target []string) bool {
	for _, v := range target {
		if obj == v {
			return true
		}
	}
	return false
}
