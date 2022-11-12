package manifests

import (
	"strings"

	lokiv1 "github.com/grafana/loki/operator/apis/loki/v1"
	"github.com/imdario/mergo"
	"github.com/operator-framework/operator-lib/proxy"
	corev1 "k8s.io/api/core/v1"
)

const (
	httpProxyKey  = "HTTP_PROXY"
	httpsProxyKey = "HTTPS_PROXY"
	noProxyKey    = "NO_PROXY"
)

func configureProxyEnv(spec *lokiv1.ClusterProxy, pod *corev1.PodSpec) error {
	if spec == nil {
		return nil
	}

	for _, envVar := range proxy.ProxyEnvNames {
		resetProxyVar(pod, envVar)
	}

	envVars := proxy.ReadProxyVarsFromEnv()
	if !spec.ReadVarsFromEnv {
		envVars = []corev1.EnvVar{
			{
				Name:  httpProxyKey,
				Value: spec.HTTPProxy,
			},
			{
				Name:  strings.ToLower(httpProxyKey),
				Value: spec.HTTPProxy,
			},
			{
				Name:  httpsProxyKey,
				Value: spec.HTTPSProxy,
			},
			{
				Name:  strings.ToLower(httpsProxyKey),
				Value: spec.HTTPSProxy,
			},
			{
				Name:  noProxyKey,
				Value: spec.NoProxy,
			},
			{
				Name:  strings.ToLower(noProxyKey),
				Value: spec.NoProxy,
			},
		}
	}

	src := corev1.Container{Env: envVars}
	for i, dst := range pod.Containers {
		if err := mergo.Merge(&dst, src, mergo.WithAppendSlice); err != nil {
			return err
		}
		pod.Containers[i] = dst
	}

	return nil
}

func resetProxyVar(podSpec *corev1.PodSpec, name string) {
	for i, container := range podSpec.Containers {
		found, index := getEnvVar(name, container.Env)
		if found {
			podSpec.Containers[i].Env = append(podSpec.Containers[i].Env[:index], podSpec.Containers[i].Env[index+1:]...)
		}
	}
}

// getEnvVar matches the given name with the envvar name
func getEnvVar(name string, envVars []corev1.EnvVar) (bool, int) {
	for i, env := range envVars {
		if env.Name == name || env.Name == strings.ToLower(name) {
			return true, i
		}
	}
	return false, 0
}
