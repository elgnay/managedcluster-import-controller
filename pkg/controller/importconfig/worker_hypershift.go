// Copyright (c) Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project

package importconfig

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"

	"github.com/stolostron/managedcluster-import-controller/pkg/constants"
	"github.com/stolostron/managedcluster-import-controller/pkg/helpers"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "open-cluster-management.io/api/cluster/v1"
	operatorv1 "open-cluster-management.io/api/operator/v1"
)

type hypershiftDetachedWorker struct {
	clientHolder *helpers.ClientHolder
}

var _ importWorker = &hypershiftDetachedWorker{}

func (w *hypershiftDetachedWorker) generateImportSecret(ctx context.Context, managedCluster *clusterv1.ManagedCluster) (*corev1.Secret, error) {
	bootStrapSecret, err := getBootstrapSecret(ctx, w.clientHolder.KubeClient, managedCluster)
	if err != nil {
		return nil, err
	}

	bootstrapKubeconfigData, err := createKubeconfigData(ctx, w.clientHolder, bootStrapSecret)
	if err != nil {
		return nil, err
	}

	imageRegistry, _, err := getImagePullSecret(ctx, w.clientHolder, managedCluster)
	if err != nil {
		return nil, err
	}

	registrationImageName, err := getImage(imageRegistry, registrationImageEnvVarName)
	if err != nil {
		return nil, err
	}

	workImageName, err := getImage(imageRegistry, workImageEnvVarName)
	if err != nil {
		return nil, err
	}

	nodeSelector, err := helpers.GetNodeSelector(managedCluster)
	if err != nil {
		return nil, err
	}

	config := KlusterletRenderConfig{
		ManagedClusterNamespace: managedCluster.Name,
		KlusterletNamespace:     klusterletNamespace,
		BootstrapKubeconfig:     base64.StdEncoding.EncodeToString(bootstrapKubeconfigData),
		RegistrationImageName:   registrationImageName,
		WorkImageName:           workImageName,
		NodeSelector:            nodeSelector,
		InstallMode:             string(operatorv1.InstallModeDetached),
	}

	files := append([]string{}, klusterletFiles...)
	importYAML := new(bytes.Buffer)
	for _, file := range files {
		template, err := manifestFiles.ReadFile(file)
		if err != nil {
			// this should not happen, if happened, panic here
			panic(err)
		}
		raw := helpers.MustCreateAssetFromTemplate(file, template, config)
		importYAML.WriteString(fmt.Sprintf("%s%s", constants.YamlSperator, string(raw)))
	}

	secret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s", managedCluster.Name, constants.ImportSecretNameSuffix),
			Namespace: managedCluster.Name,
			Labels: map[string]string{
				constants.ClusterImportSecretLabel: "",
			},
			Annotations: map[string]string{
				constants.KlusterletDeployModeAnnotation: constants.KlusterletDeployModeHypershiftDetached,
			},
		},
		Data: map[string][]byte{
			constants.ImportSecretImportYamlKey: importYAML.Bytes(),
		},
	}

	return secret, nil
}