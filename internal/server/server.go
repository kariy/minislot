// internal/server/server.go
package server

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"minislot/internal/config"
	"minislot/pkg/tiers"

	"github.com/gin-gonic/gin"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type Server struct {
	config *config.Config
	client *kubernetes.Clientset
}

type DeploymentConfig struct {
	ID           string `json:"id"`
	Namespace    string `json:"namespace"`
	Version      string `json:"version"`
	Seed         int    `json:"seed"`
	ChainID      int    `json:"chainId"`
	BlockTime    int    `json:"blockTime"`
	Tier         string `json:"tier"`
	StorageClass string `json:"storageClass"`
}

func New(cfg *config.Config, client *kubernetes.Clientset) *Server {
	return &Server{
		config: cfg,
		client: client,
	}
}

func (server *Server) Run() error {
	r  := server.Router();
	
	return r.Run(":" + server.config.Port)
}

func (server *Server) health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"alive": true,
	})
}

func (s *Server) createDeployment(c *gin.Context) {
	var config DeploymentConfig
	if err := c.BindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tier, exists := tiers.GetTier(config.Tier)
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tier"})
		return
	}

	// Create PVC
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("katana-data-%s", config.ID),
			Namespace: config.Namespace,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse(tier.PVCStorage),
				},
			},
			StorageClassName: &config.StorageClass,
		},
	}

	// Create Deployment
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("katana-%s", config.ID),
			Namespace: config.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": fmt.Sprintf("katana-%s", config.ID)},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": fmt.Sprintf("katana-%s", config.ID)},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "katana",
							Image: fmt.Sprintf("ghcr.io/dojoengine/dojo:%s", config.Version),
							Command: []string{"/bin/sh", "-c"},
							Args: []string{
								fmt.Sprintf("katana --seed=%d --chain-id=%d --block-time=%d", 
									config.Seed, config.ChainID, config.BlockTime),
							},
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceMemory: resource.MustParse(tier.MemoryRequest),
									corev1.ResourceCPU:    resource.MustParse(tier.CPURequest),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceMemory: resource.MustParse(tier.MemoryLimit),
									corev1.ResourceCPU:    resource.MustParse(tier.CPULimit),
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "katana-data",
									MountPath: "/data",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "katana-data",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: fmt.Sprintf("katana-data-%s", config.ID),
								},
							},
						},
					},
				},
			},
		},
	}

	// Create the PVC
	_, err := s.client.CoreV1().PersistentVolumeClaims(config.Namespace).Create(context.TODO(), pvc, metav1.CreateOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Error creating PVC: %v", err)})
		return
	}

	// Create the Deployment
	_, err = s.client.AppsV1().Deployments(config.Namespace).Create(context.TODO(), deployment, metav1.CreateOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Error creating Deployment: %v", err)})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Deployment created successfully"})
}

func (server *Server) Router() *gin.Engine {
    r := gin.Default()
    r.POST("/deploy", server.createDeployment)
	r.GET("/health", server.health)
    return r
}

func (server *Server) RenderTemplate(config DeploymentConfig) (string, error) {
    var renderedYAML strings.Builder
    err := server.config.Template.Execute(&renderedYAML, config)
    if err != nil {
        return "", err
    }
    return renderedYAML.String(), nil
}

func int32Ptr(i int32) *int32 { return &i }