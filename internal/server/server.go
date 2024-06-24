// internal/server/server.go
package server

import (
	"bytes"
	"context"
	"fmt"
	"net/http"

	"minislot/internal/config"

	"github.com/gin-gonic/gin"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes"
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

func (server *Server) Run() error {
	r := gin.Default()
	r.POST("/deploy", server.createDeployment)
	r.GET("/health", server.health)
	return r.Run(":" + server.config.Port)
}

func (server *Server) health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"alive": true,
	})
}

func (server *Server) createDeployment(c *gin.Context) {
	var config DeploymentConfig
	if err := c.BindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// var renderedYAML string
	var renderedYAML bytes.Buffer
	err := server.config.Template.Execute(&renderedYAML, config)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error rendering template"})
		return
	}

	scheme := runtime.NewScheme()
    codecFactory := serializer.NewCodecFactory(scheme)

    resources := []runtime.Object{}
    decoder := yaml.NewYAMLOrJSONDecoder(bytes.NewReader(renderedYAML.Bytes()), 4096)
    for {
        var rawObj runtime.RawExtension
        if err := decoder.Decode(&rawObj); err != nil {
            break
        }
        obj, _, err := codecFactory.UniversalDecoder().Decode(rawObj.Raw, nil, nil)
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Error decoding YAML"})
            return
        }
        resources = append(resources, obj)
    }


	for _, obj := range resources {
		switch o := obj.(type) {
		case *corev1.PersistentVolumeClaim:
			_, err := server.client.CoreV1().PersistentVolumeClaims(config.Namespace).Create(context.TODO(), o, metav1.CreateOptions{})
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Error creating PVC: %v", err)})
				return
			}
		case *appsv1.Deployment:
			_, err := server.client.AppsV1().Deployments(config.Namespace).Create(context.TODO(), o, metav1.CreateOptions{})
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Error creating Deployment: %v", err)})
				return
			}
		case *corev1.Service:
			_, err := server.client.CoreV1().Services(config.Namespace).Create(context.TODO(), o, metav1.CreateOptions{})
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
