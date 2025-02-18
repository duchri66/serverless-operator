package common

import (
	mf "github.com/manifestival/manifestival"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes/scheme"
)

// InjectEnvironmentIntoDeployment injects the specified environment variables into the
// specified deployment/container.
// Note: We're not deleting empty environment variables and instead set them to empty
// string. Three-way-merging of the deployment drops the update otherwise.
func InjectEnvironmentIntoDeployment(deploymentName, containerName string, envs ...corev1.EnvVar) mf.Transformer {
	return transformDeployment(deploymentName, func(deploy *appsv1.Deployment) error {
		containers := deploy.Spec.Template.Spec.Containers
		for i := range containers {
			c := &containers[i]
			if c.Name != containerName {
				continue
			}

			for _, val := range envs {
				c.Env = upsert(c.Env, val)
			}
		}

		return nil
	})
}

// upsert updates the env var if the key already exists or inserts it if it didn't
// exist.
func upsert(orgEnv []corev1.EnvVar, val corev1.EnvVar) []corev1.EnvVar {
	// Set the value if the key is already present.
	for i := range orgEnv {
		if orgEnv[i].Name == val.Name {
			orgEnv[i].Value = val.Value
			return orgEnv
		}
	}
	// If not, append a key-value pair.
	return append(orgEnv, val)
}

// transformDeployment returns a transformer that transforms a deployment with the given
// name.
func transformDeployment(name string, f func(*appsv1.Deployment) error) mf.Transformer {
	return func(u *unstructured.Unstructured) error {
		if u.GetKind() != "Deployment" || u.GetName() != name {
			return nil
		}

		deployment := &appsv1.Deployment{}
		if err := scheme.Scheme.Convert(u, deployment, nil); err != nil {
			return err
		}

		if err := f(deployment); err != nil {
			return err
		}

		if err := scheme.Scheme.Convert(deployment, u, nil); err != nil {
			return err
		}

		return nil
	}
}
