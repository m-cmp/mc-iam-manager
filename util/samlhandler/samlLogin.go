package samllogin

import (
	b64 "encoding/base64"
	"fmt"
	"log"
	"os"

	//common
	"mc_iam_manager/util/samlhandler/provider/keycloak"

	// ***** aws *****
	saml2aws "mc_iam_manager/util/samlhandler/aws/pkg"
	"mc_iam_manager/util/samlhandler/aws/pkg/awsconfig"
	awscfg "mc_iam_manager/util/samlhandler/aws/pkg/cfg"
	awscreds "mc_iam_manager/util/samlhandler/aws/pkg/creds"

	//aws-sdk
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	awssts "github.com/aws/aws-sdk-go/service/sts"

	// ***** ali *****
	saml2alibabacloud "mc_iam_manager/util/samlhandler/alibaba/pkg"
	"mc_iam_manager/util/samlhandler/alibaba/pkg/alibabacloudconfig"
	alicfg "mc_iam_manager/util/samlhandler/alibaba/pkg/cfg"
	alicreds "mc_iam_manager/util/samlhandler/alibaba/pkg/creds"

	//ali-sdk
	alists "github.com/aliyun/alibaba-cloud-sdk-go/services/sts"

	"github.com/pkg/errors"
)

// //////// AWS START
func LoginAWS(account *awscfg.IDPAccount, loginDetails *awscreds.LoginDetails) (*awsconfig.AWSCredentials, error) {
	fmt.Println("provider start")
	provider, err := keycloak.New(account)
	if err != nil {
		return nil, errors.Wrap(err, "Error building IdP client.")
	}
	fmt.Println("provider end")

	fmt.Println("samlAssertion start")
	var samlAssertion string
	samlAssertion, err = provider.Authenticate(loginDetails)
	if err != nil {
		return nil, errors.Wrap(err, "Error authenticating to IdP.")
	}
	fmt.Println("samlAssertion end")

	role, err := selectRoleAWS(samlAssertion, account)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to assume role. Please check whether you are permitted to assume the given role for the AWS service.")
	}

	awsCreds, err := loginToStsUsingRoleALIAWS(account, role, samlAssertion)
	if err != nil {
		return nil, errors.Wrap(err, "Error logging into AWS role using SAML assertion.")
	}

	return awsCreds, nil
}

func selectRoleAWS(samlAssertion string, account *awscfg.IDPAccount) (*saml2aws.AWSRole, error) {
	data, err := b64.StdEncoding.DecodeString(samlAssertion)
	if err != nil {
		return nil, errors.Wrap(err, "Error decoding SAML assertion.")
	}

	roles, err := saml2aws.ExtractAwsRoles(data)
	if err != nil {
		return nil, errors.Wrap(err, "Error parsing AWS roles.")
	}

	if len(roles) == 0 {
		log.Println("No roles to assume.")
		log.Println("Please check you are permitted to assume roles for the AWS service.")
		os.Exit(1)
	}

	awsRoles, err := saml2aws.ParseAWSRoles(roles)
	if err != nil {
		return nil, errors.Wrap(err, "Error parsing AWS roles.")
	}

	return resolveRoleALIAWS(awsRoles, samlAssertion, account)
}

func resolveRoleALIAWS(awsRoles []*saml2aws.AWSRole, samlAssertion string, account *awscfg.IDPAccount) (*saml2aws.AWSRole, error) {
	var role = new(saml2aws.AWSRole)

	if len(awsRoles) == 1 {
		if account.RoleARN != "" {
			return saml2aws.LocateRole(awsRoles, account.RoleARN)
		}
		return awsRoles[0], nil
	} else if len(awsRoles) == 0 {
		return nil, errors.New("No roles available.")
	}

	samlAssertionData, err := b64.StdEncoding.DecodeString(samlAssertion)
	if err != nil {
		return nil, errors.Wrap(err, "Error decoding SAML assertion.")
	}

	aud, err := saml2aws.ExtractDestinationURL(samlAssertionData)
	if err != nil {
		return nil, errors.Wrap(err, "Error parsing destination URL.")
	}

	awsAccounts, err := saml2aws.ParseAWSAccounts(aud, samlAssertion)
	if err != nil {
		return nil, errors.Wrap(err, "Error parsing AWS role accounts.")
	}
	if len(awsAccounts) == 0 {
		return nil, errors.New("No accounts available.")
	}

	saml2aws.AssignPrincipals(awsRoles, awsAccounts)

	if account.RoleARN != "" {
		return saml2aws.LocateRole(awsRoles, account.RoleARN)
	}
	role = awsAccounts[0].Roles[0]
	return role, nil
}

func loginToStsUsingRoleALIAWS(account *awscfg.IDPAccount, role *saml2aws.AWSRole, samlAssertion string) (*awsconfig.AWSCredentials, error) {

	sess, err := session.NewSession(&aws.Config{
		Region: &account.Region,
	})
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create session.")
	}

	svc := awssts.New(sess)

	params := &awssts.AssumeRoleWithSAMLInput{
		PrincipalArn:    aws.String(role.PrincipalARN), // Required
		RoleArn:         aws.String(role.RoleARN),      // Required
		SAMLAssertion:   aws.String(samlAssertion),     // Required
		DurationSeconds: aws.Int64(int64(account.SessionDuration)),
	}

	log.Println("Requesting AWS credentials using SAML assertion.")

	resp, err := svc.AssumeRoleWithSAML(params)
	if err != nil {
		return nil, errors.Wrap(err, "Error retrieving STS credentials using SAML.")
	}

	return &awsconfig.AWSCredentials{
		AWSAccessKey:     aws.StringValue(resp.Credentials.AccessKeyId),
		AWSSecretKey:     aws.StringValue(resp.Credentials.SecretAccessKey),
		AWSSessionToken:  aws.StringValue(resp.Credentials.SessionToken),
		AWSSecurityToken: aws.StringValue(resp.Credentials.SessionToken),
		PrincipalARN:     aws.StringValue(resp.AssumedRoleUser.Arn),
		Expires:          resp.Credentials.Expiration.Local(),
		Region:           account.Region,
	}, nil
}

////////// AWS END

// //////// ALI START
func LoginALI(account *alicfg.IDPAccount, loginDetails *alicreds.LoginDetails) (*alibabacloudconfig.AliCloudCredentials, error) {

	provider, err := saml2alibabacloud.NewSAMLClient(account)
	if err != nil {
		return nil, errors.Wrap(err, "error building IdP client")
	}

	log.Printf("Authenticating as %s ...", loginDetails.Username)

	samlAssertion, err := provider.Authenticate(loginDetails)
	if err != nil {
		return nil, errors.Wrap(err, "error authenticating to IdP")

	}

	role, err := selectRamRoleALI(samlAssertion, account)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to assume role, please check whether you are permitted to assume the given role for the AlibabaCloud STS service")
	}

	log.Println("Selected role:", role.RoleARN)

	alibabacloudCreds, err := loginToStsUsingRoleALI(account, role, samlAssertion)
	if err != nil {
		return nil, errors.Wrap(err, "error logging into AlibabaCloud role using saml assertion")
	}

	return alibabacloudCreds, nil
}

func selectRamRoleALI(samlAssertion string, account *alicfg.IDPAccount) (*saml2alibabacloud.RamRole, error) {
	data, err := b64.StdEncoding.DecodeString(samlAssertion)
	if err != nil {
		return nil, errors.Wrap(err, "error decoding saml assertion")
	}

	roles, err := saml2alibabacloud.ExtractRamRoles(data)
	if err != nil {
		return nil, errors.Wrap(err, "error parsing alicloud roles")
	}

	if len(roles) == 0 {
		log.Println("No roles to assume")
		log.Println("Please check you are permitted to assume roles for the AlibabaCloud service")
		os.Exit(1)
	}

	alibabacloudRoles, err := saml2alibabacloud.ParseRamRoles(roles)
	if err != nil {
		return nil, errors.Wrap(err, "error parsing AlibabaCloud roles")
	}

	return resolveRoleALI(alibabacloudRoles, samlAssertion, account)
}

func resolveRoleALI(alibabacloudRoles []*saml2alibabacloud.RamRole, samlAssertion string, account *alicfg.IDPAccount) (*saml2alibabacloud.RamRole, error) {
	var role = new(saml2alibabacloud.RamRole)

	if len(alibabacloudRoles) == 1 {
		if account.RoleARN != "" {
			return saml2alibabacloud.LocateRole(alibabacloudRoles, account.RoleARN)
		}
		return alibabacloudRoles[0], nil
	} else if len(alibabacloudRoles) == 0 {
		return nil, errors.New("no roles available")
	}

	samlAssertionData, err := b64.StdEncoding.DecodeString(samlAssertion)
	if err != nil {
		return nil, errors.Wrap(err, "error decoding saml assertion")
	}

	aud, err := saml2alibabacloud.ExtractDestinationURL(samlAssertionData)
	if err != nil {
		return nil, errors.Wrap(err, "error parsing destination url")
	}

	alibabacloudAccounts, err := saml2alibabacloud.ParseAlibabaCloudAccounts(aud, samlAssertion)
	if err != nil {
		return nil, errors.Wrap(err, "error parsing AlibabaCloud role accounts")
	}
	if len(alibabacloudAccounts) == 0 {
		return nil, errors.New("no accounts available")
	}

	// saml2alibabacloud.AssignPrincipals(alibabacloudRoles, alibabacloudAccounts)

	if account.RoleARN != "" {
		return saml2alibabacloud.LocateRole(alibabacloudRoles, account.RoleARN)
	}

	role = alibabacloudAccounts[0].Roles[0]

	return role, nil
}

func loginToStsUsingRoleALI(account *alicfg.IDPAccount, role *saml2alibabacloud.RamRole, samlAssertion string) (*alibabacloudconfig.AliCloudCredentials, error) {

	client, err := alists.NewClientWithAccessKey("cn-hangzhou", "saml2alibabacloud", "0.0.5")
	if err != nil {
		return nil, err
	}
	client.AppendUserAgent("saml2alibabacloud", "0.0.5")

	request := alists.CreateAssumeRoleWithSAMLRequest()
	request.Scheme = "https"
	request.RoleArn = role.RoleARN
	request.SAMLAssertion = samlAssertion
	request.SAMLProviderArn = role.PrincipalARN

	log.Println("Requesting AlibabaCloud credentials using SAML assertion")

	response, err := client.AssumeRoleWithSAML(request)
	if err != nil {
		return nil, errors.Wrap(err, "error retrieving STS credentials using SAML")
	}

	return &alibabacloudconfig.AliCloudCredentials{
		AliCloudAccessKey:     response.Credentials.AccessKeyId,
		AliCloudSecretKey:     response.Credentials.AccessKeySecret,
		AliCloudSecurityToken: response.Credentials.SecurityToken,
		PrincipalARN:          response.AssumedRoleUser.Arn,
		Region:                account.Region,
	}, nil
}

////////// ALI END
