package ossstore

import (
	"errors"
	"fmt"
	"mime"
	"path/filepath"
	"strings"
	"time"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
)

type Store struct {
	bucket *oss.Bucket
	expire int64
}

func New(endpoint, ak, sk, bucketName string, expire int64) (*Store, error) {
	if endpoint == "" || ak == "" || sk == "" || bucketName == "" {
		return nil, errors.New("OSS is not configured: set OSS_ENDPOINT, OSS_ACCESS_KEY_ID, OSS_ACCESS_KEY_SECRET, OSS_BUCKET")
	}
	client, err := oss.New(endpoint, ak, sk)
	if err != nil {
		return nil, err
	}
	bucket, err := client.Bucket(bucketName)
	if err != nil {
		return nil, err
	}
	if expire <= 0 {
		expire = 600
	}
	return &Store{bucket: bucket, expire: expire}, nil
}

func ValidateResumeFile(filename string) error {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".pdf", ".doc", ".docx":
		return nil
	default:
		return fmt.Errorf("仅支持 PDF、DOC、DOCX 简历文件，当前文件后缀为 %s", ext)
	}
}

func ResumeContentType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".pdf":
		return "application/pdf"
	case ".doc":
		return "application/msword"
	case ".docx":
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	default:
		if typ := mime.TypeByExtension(ext); typ != "" {
			return typ
		}
		return "application/octet-stream"
	}
}

func safeObjectName(filename string) string {
	base := strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename))
	var b strings.Builder
	lastUnderscore := false
	for _, r := range base {
		isSafe := (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '.'
		if isSafe {
			b.WriteRune(r)
			lastUnderscore = false
			continue
		}
		if !lastUnderscore {
			b.WriteByte('_')
			lastUnderscore = true
		}
	}
	name := strings.Trim(b.String(), "._-")
	if name == "" {
		return "resume"
	}
	return name
}

func BuildObjectKey(candidateID uint, filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	clean := safeObjectName(filename)
	return fmt.Sprintf("resumes/candidate_%d/%d_%s%s", candidateID, time.Now().UnixNano(), clean, ext)
}

func (s *Store) SignedPutURL(objectKey, contentType string) (string, error) {
	return s.bucket.SignURL(objectKey, oss.HTTPPut, s.expire, oss.ContentType(contentType))
}

func (s *Store) SignedGetURL(objectKey string) (string, error) {
	return s.bucket.SignURL(objectKey, oss.HTTPGet, s.expire)
}
