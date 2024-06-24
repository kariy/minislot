// internal/server/server.go
package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kariy/minislot/internal/config"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/yaml"
)

type Server struct {
	config *config.Config
	client *kubernetes.Clientset
}

type DeploymentConfig struct {
	ID             string `json:"id"`
	Namespace      string `json:"namespace"`
	Version        string `json:"version"`
	Seed           int    `json:"seed"`
	ChainID        int    `json:"chainId"`
	BlockTime      int    `json:"blockTime"`
	Storage        int    `json:"storage"`
	StorageClass   string `json:"storageClass"`
	Resources      struct {
		Requests struct {
			Memory string `json:"memory"`
			CPU    string `json:"cpu"`
		} `json:"requests"`
		Limits struct {
			Memory string `json:"memory"`
			CPU    string `json:"cpu"`
		} `json:"limits"`
	} `json:"resources"`
}

func New(cfg *config.Config, client *kubernetes.Clientset) *Server {
	return &Server{
		config: cfg,
		client: client,
	}
}

func (s *Server) Run() error {
	r := gin.Default()
	r.POST("/deploy", s.createDeployment)
	return r.Run(":" + s.config.Port)
}

func (s *Server) createDeployment(c *gin.Context) {
	var config DeploymentConfig
	if err := c.BindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var renderedYAML string
	err := s.config.Template.Execute(&renderedYAML, config)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error rendering template"})
		return
	}

	resources := []runtime.Object{}
	decoder := yaml.NewYAMLOrJSONDecoder([]byte(renderedYAML), 4096)
	for {
		var rawObj runtime.RawExtension
		if err := decoder.Decode(&rawObj); err != nil {
			break
		}
		obj, _, err := runtime.UnstructuredJSONScheme.Decode(rawObj.Raw, nil, nil)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error decoding YAML"})
			return
		}
		resources = append(resources, obj)
	}

	for _, obj := range resources {
		switch o := obj.(type) {
		case *corev1.PersistentVolumeClaim:
			_, err := s.client.CoreV1().PersistentVolumeClaims(config.Namespace).Create(context.TODO(), o, metav1.CreateOptions{})
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Error creating PVC: %v", err)})
				return
			}
		case *appsv1.Deployment:
			_, err := s.client.AppsV1().Deployments(config.Namespace).Create(context.TODO(), o, metav1.CreateOptions{})
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Error creating Deployment: %v", err)})
				return
			}
		case *corev1.Service:
			_, err := s.client.CoreV1().Services(config.Namespace).Create(context.TODO(), o, metav1.CreateOptions{})
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Error creating Service: %v", err)})
				return
			}
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Unknown resource type"})
			return
		}
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Deployment created successfully"})
}
