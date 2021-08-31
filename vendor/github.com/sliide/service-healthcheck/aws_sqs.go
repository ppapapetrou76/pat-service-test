package healthcheck

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sts"
)

const (
	AWSSQSPermissionGetAttributes int = 1 << iota
	AWSSQSPermissionSendMessage
	AWSSQSPermissionReceiveMessage
	AWSSQSPermissionDeleteMessage
	AWSSQSPermissionPurgeQueue
	AWSSQSPermissionChangeMessageVisibility
	AWSSQSPermissionSetQueueAttributes
	AWSSQSPermissionTagQueue
	AWSSQSPermissionUntagQueue
	AWSSQSPermissionListDeadLetterSourceQueues
	AWSSQSPermissionListQueueTags
)

// AWSSQSPermissionCheck returns a function that checks the AWS SQS permission by the given flags
func AWSSQSPermissionCheck(queueSession *session.Session, queueURL string, permissionFlags int) CheckingFunc {
	if queueSession == nil {
		return func(context.Context) (*CheckingState, error) {
			return nil, errors.New("queueSession is nil")
		}
	}

	var region string
	if cfg := queueSession.Config; cfg != nil {
		region = aws.StringValue(cfg.Region)
	}

	auth := &awsAuthority{
		region: region,
		sqs:    sqs.New(queueSession),
		sts:    sts.New(queueSession),
		iam:    iam.New(queueSession),
	}

	actions := sqsActionsFromPermissionFlags(permissionFlags)

	return func(ctx context.Context) (*CheckingState, error) {

		callerARN, err := auth.QueryCallerARN(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to query caller arn: %w", err)
		}

		roleARN, ok := getRoleARN(callerARN)
		if !ok {
			return nil, fmt.Errorf("failed to get role arn from the caller arn '%s': %w", callerARN, err)
		}

		queueARN, err := auth.QueryQueueARN(ctx, queueURL)
		if err != nil {
			return nil, fmt.Errorf("failed to query queue arn: %w", err)
		}

		evalResults, err := auth.SimulatePolicy(ctx, roleARN, queueARN, actions)
		if err != nil {
			return nil, fmt.Errorf("failed to simulate policy: %w", err)
		}

		denyActions := make([]string, 0, len(actions))
		for _, e := range evalResults {
			evalActionName := aws.StringValue(e.EvalActionName)
			evalActionDecision := aws.StringValue(e.EvalDecision)

			if evalActionDecision != iam.PolicyEvaluationDecisionTypeAllowed {
				denyActions = append(denyActions, evalActionName)
			}
		}

		if len(denyActions) <= 0 {
			return &CheckingState{
				State:  StateHealthy,
				Output: fmt.Sprintf("Have full permission: %v", actions),
			}, nil
		}
		return &CheckingState{
			State:  StateUnhealthy,
			Output: fmt.Sprintf("Not enough permission: %v", denyActions),
		}, nil
	}
}

func getRoleARN(arn string) (string, bool) {
	if strings.HasPrefix(arn, "arn:aws:iam::") && strings.Contains(arn, ":user/") {
		// User role, such `arn:aws:iam::410715645895:user/siyuan`
		return arn, true
	}

	// Convert the assume-role arn to role arn if possible
	// arn:aws:sts::410715645895:assumed-role/k8-notification-service-dev/c69008ce-k8-notification-service-dev
	// arn:aws:iam::410715645895:role/k8-notification-service-dev
	regexp := regexp.MustCompile(`arn:aws:sts::(.*?):assumed-role/(.*?)/`)
	ss := regexp.FindStringSubmatch(arn)
	if ss == nil {
		return "", false
	}
	return fmt.Sprintf("arn:aws:iam::%s:role/%s", ss[1], ss[2]), true
}

func sqsActionsFromPermissionFlags(flags int) []string {
	actions := make([]string, 0)

	// For the detail of permissions, please ref to the following link
	// https://docs.aws.amazon.com/AWSSimpleQueueService/latest/SQSDeveloperGuide/sqs-api-permissions-reference.html
	if (flags & AWSSQSPermissionGetAttributes) > 0 {
		actions = append(actions, "sqs:GetQueueAttributes")
	}
	if (flags & AWSSQSPermissionSendMessage) > 0 {
		actions = append(actions, "sqs:SendMessage")
	}
	if (flags & AWSSQSPermissionReceiveMessage) > 0 {
		actions = append(actions, "sqs:ReceiveMessage")
	}
	if (flags & AWSSQSPermissionDeleteMessage) > 0 {
		actions = append(actions, "sqs:DeleteMessage")
	}
	if (flags & AWSSQSPermissionPurgeQueue) > 0 {
		actions = append(actions, "sqs:PurgeQueue")
	}
	if (flags & AWSSQSPermissionChangeMessageVisibility) > 0 {
		actions = append(actions, "sqs:ChangeMessageVisibility")
	}
	if (flags & AWSSQSPermissionSetQueueAttributes) > 0 {
		actions = append(actions, "sqs:SetQueueAttributes")
	}
	if (flags & AWSSQSPermissionTagQueue) > 0 {
		actions = append(actions, "sqs:TagQueue")
	}
	if (flags & AWSSQSPermissionUntagQueue) > 0 {
		actions = append(actions, "sqs:UntagQueue")
	}
	if (flags & AWSSQSPermissionListDeadLetterSourceQueues) > 0 {
		actions = append(actions, "sqs:ListDeadLetterSourceQueues")
	}
	if (flags & AWSSQSPermissionListQueueTags) > 0 {
		actions = append(actions, "sqs:ListQueueTags")
	}
	return actions
}

type awsAuthority struct {
	region string

	sts *sts.STS
	sqs *sqs.SQS
	iam *iam.IAM
}

func (c awsAuthority) Region() string {
	return c.region
}

func (c awsAuthority) QueryInstanceARN(ctx aws.Context) (string, error) {
	// Try to get instance arn by the metadata service,
	// please ref to the links for the detail:
	// for EC2: https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/instancedata-data-retrieval.html
	// for EKS: https://docs.aws.amazon.com/eks/latest/userguide/restrict-ec2-credential-access.html
	// the format: https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/instancedata-data-categories.html
	//
	request, err := http.NewRequest(http.MethodGet, "http://169.254.169.254/latest/meta-data/iam/info", nil)
	if err != nil {
		return "", fmt.Errorf("failed to build request: %w", err)
	}

	request = request.WithContext(ctx)

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return "", fmt.Errorf("failed to query from metadata service: %w", err)
	}

	defer response.Body.Close()
	decoder := json.NewDecoder(response.Body)

	var m map[string]string
	if err := decoder.Decode(&m); err != nil {
		return "", fmt.Errorf("failed to decode the metadata: %w", err)
	}

	arn, ok := m["InstanceProfileArn"]
	if !ok {
		return "", fmt.Errorf("cannot find 'InstanceProfileArn' in the metadata")
	}
	return arn, nil
}

func (c awsAuthority) QueryCallerARN(ctx aws.Context) (string, error) {

	response, err := c.sts.GetCallerIdentityWithContext(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return "", nil
	}
	return aws.StringValue(response.Arn), nil
}

func (c awsAuthority) QueryQueueARN(ctx aws.Context, url string) (string, error) {
	response, err := c.sqs.GetQueueAttributesWithContext(ctx, &sqs.GetQueueAttributesInput{
		AttributeNames: aws.StringSlice([]string{
			sqs.QueueAttributeNameQueueArn,
		}),
		QueueUrl: aws.String(url),
	})
	if err != nil {
		return "", err
	}

	arn := aws.StringValue(response.Attributes[sqs.QueueAttributeNameQueueArn])
	if arn == "" {
		return "", fmt.Errorf("empty amazon resource name")
	}
	return arn, nil
}

func (c awsAuthority) SimulatePolicy(ctx aws.Context, callerARN, resourceARN string, actions []string) ([]*iam.EvaluationResult, error) {
	if len(resourceARN) <= 0 || len(actions) <= 0 {
		return []*iam.EvaluationResult{}, nil
	}

	evalResults := make([]*iam.EvaluationResult, 0)
	err := c.iam.SimulatePrincipalPolicyPagesWithContext(ctx, &iam.SimulatePrincipalPolicyInput{
		ActionNames:     aws.StringSlice(actions),
		PolicySourceArn: aws.String(callerARN), // PolicySourceArn works with ROLE or USER, unlike CallerArn only works with USER
		ResourceArns:    aws.StringSlice([]string{resourceARN}),
	}, func(response *iam.SimulatePolicyResponse, lastPage bool) bool {
		evalResults = append(evalResults, response.EvaluationResults...)
		return !lastPage // Return false to stop this operation
	})
	if err != nil {
		return nil, err
	}

	return evalResults, nil
}
