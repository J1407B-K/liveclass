package cos

import (
	"context"
	"fmt"
	"github.com/tencentyun/cos-go-sdk-v5"
	"os"
)

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
