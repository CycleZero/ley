package infra

//func NewMinioClient(
//	endpoint string,
//	accessKeyID string,
//	secretAccessKey string,
//	bucketName string,
//) *minio.Client {
//
//	// Initialize minio client object.
//	minioClient, err := minio.New(endpoint, &minio.Options{
//		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
//		Secure: false,
//	})
//	if err != nil {
//		log.GetLogger().Error("Error creating minio client", zap.Error(err))
//		panic(err)
//	}
//
//	ctx := context.Background()
//	err = minioClient.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{})
//	if err != nil {
//		// Check to see if we already own this bucket (which happens if you run this twice)
//		exists, errBucketExists := minioClient.BucketExists(ctx, bucketName)
//		if errBucketExists == nil && exists {
//			log.GetLogger().Sugar().Infof("We already own %s\n", bucketName)
//		} else {
//			log.GetLogger().Sugar().Fatal(err)
//		}
//	} else {
//		log.GetLogger().Sugar().Infof("Successfully created %s\n", bucketName)
//	}
//	return minioClient
//}
