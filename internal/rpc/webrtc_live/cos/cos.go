package cos

import (
	"context"
	"fmt"
	"github.com/tencentyun/cos-go-sdk-v5"
	"os"
)

type myProgressListener struct{}

func (l *myProgressListener) ProgressChangedCallback(event *cos.ProgressEvent) {
	fmt.Printf("下载进度：%d/%d bytes\n", event.ConsumedBytes, event.TotalBytes)
}

func UploadToCos(ctx context.Context, cosClient *cos.Client, localFile, lessonId, filename string) error {
	cosPath := fmt.Sprintf("lesson_%s/%s", lessonId, filename)

	file, err := os.Open(localFile)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = cosClient.Object.Put(ctx, cosPath, file, nil)
	if err != nil {
		return fmt.Errorf("上传COS失败: %w", err)
	}
	return nil
}

func DownloadFromCos(ctx context.Context, cosClient *cos.Client, localFile, key string) ([]byte, error) {
	opt := &cos.MultiDownloadOptions{
		PartSize:        5,
		ThreadPoolSize:  4,
		CheckPoint:      true,
		DisableChecksum: false,
		Opt: &cos.ObjectGetOptions{
			Listener: &myProgressListener{},
		},
	}
	resp, err := cosClient.Object.Download(ctx, key, localFile, opt)
	if err != nil {
		return nil, fmt.Errorf("下载COS失败: %w", err)
	}
	defer resp.Body.Close()

	data, err := os.ReadFile(localFile)
	if err != nil {
		return nil, fmt.Errorf("读取临时文件失败: %w", err)
	}

	return data, nil
}
