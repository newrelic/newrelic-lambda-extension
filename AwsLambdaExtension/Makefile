# ----- Target used with SAM (Serverless Application Model) -----
build-AwsLambdaExtension:
	echo :build-AwsLambdaExtension
	GOOS=linux GOARCH=amd64 go build -o $(ARTIFACTS_DIR)/extensions/AwsLambdaExtension main.go
	chmod +x $(ARTIFACTS_DIR)/extensions/AwsLambdaExtension

# ----- Target illustrating manual steps required to create the extension layer -----
buildAndDeployExtensionLayer:
	echo :buildAndDeployExtensionLayer
	rm -rf bin
	GOOS=linux GOARCH=amd64 go build -o bin/extensions/AwsLambdaExtension main.go
	chmod +x bin/extensions/AwsLambdaExtension
	cd bin && zip -r extension.zip extensions

	aws lambda publish-layer-version \
		--layer-name "AwsLambdaExtension" \
		--zip-file "fileb://bin/extension.zip"
