package manifests

import (
	"strings"

	"github.com/imdario/mergo"
	corev1 "k8s.io/api/core/v1"
)

const (
	httpProxyKey  = "HTTP_PROXY"
	httpsProxyKey = "HTTPS_PROXY"
	noProxyKey    = "NO_PROXY"
)

var (
	ProxyEnvNames = []string{
		httpProxyKey,
		strings.ToLower(httpProxyKey),
		httpsProxyKey,
		strings.ToLower(httpsProxyKey),
		noProxyKey,
		strings.ToLower(noProxyKey),
	}
)

func configureProxyEnv(pod *corev1.PodSpec, opts Options) error {
	for _, envVar := range ProxyEnvNames {
		resetProxyVar(pod, envVar)
	}

	envVars := opts.EnvVars
	if envVars == nil {
		spec := opts.Stack.Proxy
		if spec == nil {
			return nil
		}

		envVars = []corev1.EnvVar{}

		if spec.HTTPProxy != "" {
			envVars = append(envVars,
				corev1.EnvVar{
					Name:  httpProxyKey,
					Value: spec.HTTPProxy,
				},
				corev1.EnvVar{
					Name:  strings.ToLower(httpProxyKey),
					Value: spec.HTTPProxy,
				},
			)
		}

		if spec.HTTPSProxy != "" {
			envVars = append(envVars,
				corev1.EnvVar{
					Name:  httpsProxyKey,
					Value: spec.HTTPSProxy,
				},
				corev1.EnvVar{
					Name:  strings.ToLower(httpsProxyKey),
					Value: spec.HTTPSProxy,
				},
			)
		}

		if spec.NoProxy != "" {
			envVars = append(envVars,
				corev1.EnvVar{
					Name:  noProxyKey,
					Value: spec.NoProxy,
				},
				corev1.EnvVar{
					Name:  strings.ToLower(noProxyKey),
					Value: spec.NoProxy,
				},
			)
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
