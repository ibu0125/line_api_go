package azure

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/sas"
)

var(
	AZURE_STORAGE_ACCOUNT=os.Getenv("AZURE_STORAGE_ACCOUNT")
	AZURE_STORAGE_KEY=os.Getenv("AZURE_STORAGE_KEY")
)


func UploadDocx(container, blobName, localPath string) error {


	cred,err:=azblob.NewSharedKeyCredential(AZURE_STORAGE_ACCOUNT,AZURE_STORAGE_KEY)
	if err!=nil {
		return err
	}

	serviceURL:=fmt.Sprintf("https://%s.blob.core.windows.net/",AZURE_STORAGE_ACCOUNT)

	client,err:=azblob.NewClientWithSharedKeyCredential(serviceURL,cred,nil)
	if err != nil {
    	return err
	}

    f, err := os.Open(localPath)
    if err != nil {
        return err
    }
    defer f.Close()

    _, err = client.UploadFile(
        context.Background(),
        container,    // コンテナ名
        blobName,     // BLOB名
        f,            // *os.File
        &azblob.UploadFileOptions{
            HTTPHeaders: &blob.HTTPHeaders{ // ← azblob.HTTPHeaders ではなく blob.HTTPHeaders
                BlobContentType: to.Ptr("application/vnd.openxmlformats-officedocument.wordprocessingml.document"),
            },
        },
    )
  
	return err
}

func GenerateBlobSASURL(containerName,blobName string,expireMinutes int)(string,error){
	cred,err:=azblob.NewSharedKeyCredential(AZURE_STORAGE_ACCOUNT,AZURE_STORAGE_KEY)
	if err!=nil {
		return "",err
	}

	permissions:=sas.BlobPermissions{
		Read: true,
	}

	startTime:=time.Now().Add(-5*time.Minute)
	expireTime:=time.Now().Add(time.Duration(expireMinutes)*time.Minute)


	sasQueryParams,err:=sas.BlobSignatureValues{
		Protocol: sas.ProtocolHTTPS,
		StartTime: startTime,
		ExpiryTime: expireTime,
		Permissions: permissions.String(),
		ContainerName: containerName,
		BlobName: blobName,
	}.SignWithSharedKey(cred)
	if err!=nil {
		return "",err
	}

	sasURL:=fmt.Sprintf(
		"https://%s.blob.core.windows.net/%s/%s?%s",
		AZURE_STORAGE_ACCOUNT,
		containerName,
		blobName,
		sasQueryParams.Encode(),
	)

	return sasURL,nil
}