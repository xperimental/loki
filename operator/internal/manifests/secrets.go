package manifests

import (
	"github.com/grafana/loki/operator/internal/manifests/storage"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func CreateStorageSecret(objectStorage storage.Options, namespace string, name string) *corev1.Secret {

	/* switch storageType := objectStorage.StorageType; storageType {
	case lokiv1.ObjectStorageSecretAzure:

	case lokiv1.ObjectStorageSecretGCS:

	case lokiv1.ObjectStorageSecretS3:
		//if objectStorage.S3.SSE
		Data := map[string][]byte{
			"storage":           []byte("S3"),
			"endpoint":          []byte(objectStorage.S3.Endpoint),
			"region":            []byte(objectStorage.S3.Region),
			"bucketnames":       []byte(objectStorage.S3.Buckets),
			"access_key_id":     []byte(objectStorage.S3.AccessKeyID),
			"access_key_secret": []byte(objectStorage.S3.AccessKeySecret),
			"SSE":               []byte(""),
		}
	case lokiv1.ObjectStorageSecretSwift:

	case lokiv1.ObjectStorageSecretAlibabaCloud:

	default:
		return nil, kverrors.New("unknown storage type", "type", secretType)
	} */

	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: corev1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"storage":           []byte("S3"),
			"endpoint":          []byte(objectStorage.S3.Endpoint),
			"region":            []byte(objectStorage.S3.Region),
			"bucketnames":       []byte(objectStorage.S3.Buckets),
			"access_key_id":     []byte(objectStorage.S3.AccessKeyID),
			"access_key_secret": []byte(objectStorage.S3.AccessKeySecret),
		},
	}
}
