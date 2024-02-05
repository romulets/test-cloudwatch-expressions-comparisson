package cloudwatch_lep

import (
	"errors"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestParse(t *testing.T) {
	cases := map[string]struct {
		in  string
		out expression
		err error
	}{
		"simple expression": {
			in:  "{$.eventName=DeleteGroupPolicy}",
			out: se("$.eventName", coEqual, "DeleteGroupPolicy"),
		},
		"simple expression with spaces": {
			in:  "{   $.eventName = DeleteGroupPolicy   }",
			out: se("$.eventName", coEqual, "DeleteGroupPolicy"),
		},
		"simple expression with spaces in the middle": {
			in:  "{   $. eventName = DeleteGroupPolicy   }",
			out: se("$. eventName", coEqual, "DeleteGroupPolicy"),
		},
		"simple expression with string": {
			in:  "{   $. eventName = \" String string string  \" }",
			out: se("$. eventName", coEqual, "\" String string string  \""),
		},
		"simple expression 'different' comparator": {
			in:  "{   $. eventName != \" String string string  \" }",
			out: se("$. eventName", coNotEqual, "\" String string string  \""),
		},
		"simple expression 'notExists' comparator": {
			in:  "{   $.eventName NOT EXISTS }",
			out: se("$.eventName", coNotExists, ""),
		},
		"simple expression with parenthesis": {
			in:  "{($.eventName=DeleteGroupPolicy)}",
			out: se("$.eventName", coEqual, "DeleteGroupPolicy"),
		},
		"simple expression with multiple parenthesis": {
			in:  "{(($.eventName=DeleteGroupPolicy))}",
			out: se("$.eventName", coEqual, "DeleteGroupPolicy"),
		},
		"simple expression with parenthesis and spaces": {
			in:  "{   (   $.eventName  =   DeleteGroupPolicy )   }",
			out: se("$.eventName", coEqual, "DeleteGroupPolicy"),
		},
		"error on broken parenthesis and spaces": {
			in:  "{   (   $.eventName  =   DeleteGroupPolicy ))   }",
			err: errors.New("broken parenthesis"),
			out: simpleExpression{},
		},
		"error on double operators (double equals)": {
			in:  "{   $.eventName == a }",
			err: errors.New("got multiple comparison operators"),
			out: simpleExpression{},
		},
		"error on double operators (different and equals)": {
			in:  "{   $.eventName !== a }",
			err: errors.New("got multiple comparison operators"),
			out: simpleExpression{},
		},
		"error on double operators (after expression)": {
			in:  "{   $.eventName != a !=}",
			err: errors.New("got multiple comparison operators"),
			out: simpleExpression{},
		},
		"complex expression 2 expressions": {
			in: "{$.userIdentity.type = \"Root\" && $.userIdentity.invokedBy NOT EXISTS}",
			out: ce("&&",
				se("$.userIdentity.type", coEqual, "\"Root\""),
				se("$.userIdentity.invokedBy", coNotExists, "")),
		},
		"complex expression 2 expressions with outer parenthesis": {
			in: "{($.userIdentity.type = \"Root\" && $.userIdentity.invokedBy NOT EXISTS)}",
			out: ce("&&",
				se("$.userIdentity.type", coEqual, "\"Root\""),
				se("$.userIdentity.invokedBy", coNotExists, "")),
		},
		"complex expression with parenthesis per simple expression": {
			in: "{($.errorCode = \"*UnauthorizedOperation\") || ($.errorCode = \"AccessDenied*\") || ($.sourceIPAddress!=\"delivery.logs.amazonaws.com\") || ($.eventName!=\"HeadBucket\") }",
			out: ce("||",
				se("$.errorCode", coEqual, "\"*UnauthorizedOperation\""),
				se("$.errorCode", coEqual, "\"AccessDenied*\""),
				se("$.sourceIPAddress", coNotEqual, "\"delivery.logs.amazonaws.com\""),
				se("$.eventName", coNotEqual, "\"HeadBucket\""),
			),
		},
		"complex expression 3 expressions": {
			in: "{$.userIdentity.type = \"Root\" && $.userIdentity.invokedBy NOT EXISTS && $.eventType != \"AwsServiceEvent\" }",
			out: ce("&&",
				se("$.userIdentity.type", coEqual, "\"Root\""),
				se("$.userIdentity.invokedBy", coNotExists, ""),
				se("$.eventType", coNotEqual, "\"AwsServiceEvent\""),
			),
		},
		"complex expression 2 logical operators": {
			in: "{($.eventSource = kms.amazonaws.com) && (($.eventName=DisableKey)||($.eventName=ScheduleKeyDeletion)) }",
			out: ce("&&",
				se("$.eventSource", coEqual, "kms.amazonaws.com"),
				ce("||",
					se("$.eventName", coEqual, "DisableKey"),
					se("$.eventName", coEqual, "ScheduleKeyDeletion"),
				),
			),
		},
		"sub expression first": {
			in: "{ (($.eventName=DisableKey)||($.eventName=ScheduleKeyDeletion)) && ($.eventSource = kms.amazonaws.com) }",
			out: ce("&&",
				ce("||",
					se("$.eventName", coEqual, "DisableKey"),
					se("$.eventName", coEqual, "ScheduleKeyDeletion"),
				),
				se("$.eventSource", coEqual, "kms.amazonaws.com"),
			),
		},
		"error on complex expression alternating logical operators": {
			in:  "{($.eventSource = kms.amazonaws.com) && ($.eventName=DisableKey) || ($.eventName=ScheduleKeyDeletion)}",
			err: errors.New("not supported comparison with alternating logical operators"),
			out: simpleExpression{},
		},
		"4 layers deep expression": {
			in:  "{((a=b) && ((a=b) || ((a=b) && (a!=b || (a=b)))))}",
			err: errors.New("not supported more than 2 nested parenthesis"),
			out: simpleExpression{},
		},
		//"too deep expression": {
		//	in:  "{((((((((((((a=b)&&(a=b))&&(a=b))&&(a=b))&&(a=b))&&(a=b))&&(a=b))&&(a=b))&&(a=b))&&(a=b))&&(a=b))&&(a=b))}",
		//	err: errors.New("not supported comparison with alternating logical operators"),
		//	out: simpleExpression{},
		//},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			s, err := parse(tc.in)
			require.Equal(t, tc.err, err)
			require.Equal(t, tc.out, s)
		})
	}
}

func TestSimpleExpression_isEquivalent(t *testing.T) {
	cases := map[string]struct {
		a   expression
		b   expression
		out bool
	}{
		"same expression": {
			a:   se("a", coEqual, "b"),
			b:   se("a", coEqual, "b"),
			out: true,
		},
		"invert expression": {
			a:   se("a", coEqual, "b"),
			b:   se("b", coEqual, "a"),
			out: true,
		},
		"special chars": {
			a:   se("\"!@#$%ˆ&*()\"", coEqual, "\">P{?}|     }}{|\""),
			b:   se("\">P{?}|     }}{|\"", coEqual, "\"!@#$%ˆ&*()\""),
			out: true,
		},
		"operator not exists": {
			a:   se("a", coNotExists, "b"),
			b:   se("b", coNotExists, "a"),
			out: true,
		},
		"operator different": {
			a:   se("a", coNotEqual, "b"),
			b:   se("b", coNotEqual, "a"),
			out: true,
		},
		"operator doesn't match": {
			a:   se("a", coNotEqual, "b"),
			b:   se("b", coEqual, "a"),
			out: false,
		},
		"values doesn't match on right": {
			a:   se("a", coNotEqual, "b"),
			b:   se("a", coNotEqual, "DIFF"),
			out: false,
		},
		"values doesn't match on left": {
			a:   se("a", coNotEqual, "b"),
			b:   se("DIFF", coNotEqual, "b"),
			out: false,
		},
		"values doesn't match on right inverted": {
			a:   se("a", coNotEqual, "b"),
			b:   se("DIFF", coNotEqual, "a"),
			out: false,
		},
		"values doesn't match on both": {
			a:   se("a", coNotEqual, "b"),
			b:   se("DIFF", coNotEqual, "DIFF2"),
			out: false,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, tc.out, tc.a.isEquivalent(tc.b))
			require.Equal(t, tc.out, tc.b.isEquivalent(tc.a))
		})
	}
}

func TestComplexExpression_isEquivalent(t *testing.T) {
	cases := map[string]struct {
		a   expression
		b   expression
		out bool
	}{
		"same expression": {
			a: ce("&&",
				se("$.userIdentity.type", coEqual, "\"Root\""),
				se("$.userIdentity.invokedBy", coNotExists, ""),
				se("$.eventType", coNotEqual, "\"AwsServiceEvent\""),
			),
			b: ce("&&",
				se("$.userIdentity.type", coEqual, "\"Root\""),
				se("$.userIdentity.invokedBy", coNotExists, ""),
				se("$.eventType", coNotEqual, "\"AwsServiceEvent\""),
			),
			out: true,
		},
		"different order": {
			a: ce("&&",
				se("$.eventType", coNotEqual, "\"AwsServiceEvent\""),
				se("\"Root\"", coEqual, "$.userIdentity.type"),
				se("$.userIdentity.invokedBy", coNotExists, ""),
			),
			b: ce("&&",
				se("$.userIdentity.invokedBy", coNotExists, ""),
				se("$.userIdentity.type", coEqual, "\"Root\""),
				se("$.eventType", coNotEqual, "\"AwsServiceEvent\""),
			),
			out: true,
		},
		"same sub expressions": {
			a: ce("&&",
				se("$.eventSource", coEqual, "kms.amazonaws.com"),
				ce("||",
					se("$.eventName", coEqual, "DisableKey"),
					se("$.eventName", coEqual, "ScheduleKeyDeletion"),
				),
			),
			b: ce("&&",
				se("$.eventSource", coEqual, "kms.amazonaws.com"),
				ce("||",
					se("$.eventName", coEqual, "DisableKey"),
					se("$.eventName", coEqual, "ScheduleKeyDeletion"),
				),
			),
			out: true,
		},
		"sub expressions different order": {
			a: ce("&&",
				ce("||",
					se("$.eventName", coEqual, "DisableKey"),
					se("$.eventName", coEqual, "ScheduleKeyDeletion"),
				),
				se("$.eventSource", coEqual, "kms.amazonaws.com"),
			),
			b: ce("&&",
				se("$.eventSource", coEqual, "kms.amazonaws.com"),
				ce("||",
					se("$.eventName", coEqual, "ScheduleKeyDeletion"),
					se("$.eventName", coEqual, "DisableKey"),
				),
			),
			out: true,
		},
		"different logical operator": {
			a: ce("&&",
				se("$.userIdentity.type", coEqual, "\"Root\""),
				se("$.userIdentity.invokedBy", coNotExists, ""),
				se("$.eventType", coNotEqual, "\"AwsServiceEvent\""),
			),
			b: ce("||",
				se("$.userIdentity.type", coEqual, "\"Root\""),
				se("$.userIdentity.invokedBy", coNotExists, ""),
				se("$.eventType", coNotEqual, "\"AwsServiceEvent\""),
			),
			out: false,
		},
		"one not matching": {
			a: ce("&&",
				se("$.userIdentity.type", coEqual, "\"Rootty\""),
				se("$.userIdentity.invokedBy", coNotExists, ""),
				se("$.eventType", coNotEqual, "\"AwsServiceEvent\""),
			),
			b: ce("&&",
				se("$.userIdentity.type", coEqual, "\"Root\""),
				se("$.userIdentity.invokedBy", coNotExists, ""),
				se("$.eventType", coNotEqual, "\"AwsServiceEvent\""),
			),
			out: false,
		},
		"one not missing": {
			a: ce("&&",
				se("$.userIdentity.invokedBy", coNotExists, ""),
				se("$.eventType", coNotEqual, "\"AwsServiceEvent\""),
			),
			b: ce("&&",
				se("$.userIdentity.type", coEqual, "\"Root\""),
				se("$.userIdentity.invokedBy", coNotExists, ""),
				se("$.eventType", coNotEqual, "\"AwsServiceEvent\""),
			),
			out: false,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, tc.out, tc.a.isEquivalent(tc.b))
			require.Equal(t, tc.out, tc.b.isEquivalent(tc.a))
		})
	}
}

func TestAreCloudWatchExpressionsEquivalent(t *testing.T) {
	cases := map[string]struct {
		expA               string
		expB               string
		shouldBeEquivalent bool
		err                error
	}{
		"Same Expressions [1]": {
			expA:               "{ ($.errorCode = \"*UnauthorizedOperation\") || ($.errorCode = \"AccessDenied*\") || ($.sourceIPAddress!=\"delivery.logs.amazonaws.com\") || ($.eventName!=\"HeadBucket\") }",
			expB:               "{ ($.errorCode = \"*UnauthorizedOperation\") || ($.errorCode = \"AccessDenied*\") || ($.sourceIPAddress!=\"delivery.logs.amazonaws.com\") || ($.eventName!=\"HeadBucket\") }",
			shouldBeEquivalent: true,
		},

		"Same Expressions [2]": {
			expA:               "{ ($.eventName = CreateNetworkAcl) || ($.eventName = CreateNetworkAclEntry) || ($.eventName = DeleteNetworkAcl) || ($.eventName = DeleteNetworkAclEntry) || ($.eventName = ReplaceNetworkAclEntry) || ($.eventName = ReplaceNetworkAclAssociation) }",
			expB:               "{ ($.eventName = CreateNetworkAcl) || ($.eventName = CreateNetworkAclEntry) || ($.eventName = DeleteNetworkAcl) || ($.eventName = DeleteNetworkAclEntry) || ($.eventName = ReplaceNetworkAclEntry) || ($.eventName = ReplaceNetworkAclAssociation) }",
			shouldBeEquivalent: true,
		},

		"Same Expressions [3]": {
			expA:               "{ ($.eventName = CreateCustomerGateway) || ($.eventName = DeleteCustomerGateway) || ($.eventName = AttachInternetGateway) || ($.eventName = CreateInternetGateway) || ($.eventName = DeleteInternetGateway) || ($.eventName = DetachInternetGateway) }",
			expB:               "{ ($.eventName = CreateCustomerGateway) || ($.eventName = DeleteCustomerGateway) || ($.eventName = AttachInternetGateway) || ($.eventName = CreateInternetGateway) || ($.eventName = DeleteInternetGateway) || ($.eventName = DetachInternetGateway) }",
			shouldBeEquivalent: true,
		},

		"Same Expressions [4]": {
			expA:               "{ ($.eventName = CreateRoute) || ($.eventName = CreateRouteTable) || ($.eventName = ReplaceRoute) || ($.eventName = ReplaceRouteTableAssociation) || ($.eventName = DeleteRouteTable) || ($.eventName = DeleteRoute) || ($.eventName = DisassociateRouteTable) }",
			expB:               "{ ($.eventName = CreateRoute) || ($.eventName = CreateRouteTable) || ($.eventName = ReplaceRoute) || ($.eventName = ReplaceRouteTableAssociation) || ($.eventName = DeleteRouteTable) || ($.eventName = DeleteRoute) || ($.eventName = DisassociateRouteTable) }",
			shouldBeEquivalent: true,
		},

		"Same Expressions [5]": {
			expA:               "{ ($.eventName = CreateVpc) || ($.eventName = DeleteVpc) || ($.eventName = ModifyVpcAttribute) || ($.eventName = AcceptVpcPeeringConnection) || ($.eventName = CreateVpcPeeringConnection) || ($.eventName = DeleteVpcPeeringConnection) || ($.eventName = RejectVpcPeeringConnection) || ($.eventName = AttachClassicLinkVpc) || ($.eventName = DetachClassicLinkVpc) || ($.eventName = DisableVpcClassicLink) || ($.eventName = EnableVpcClassicLink) }",
			expB:               "{ ($.eventName = CreateVpc) || ($.eventName = DeleteVpc) || ($.eventName = ModifyVpcAttribute) || ($.eventName = AcceptVpcPeeringConnection) || ($.eventName = CreateVpcPeeringConnection) || ($.eventName = DeleteVpcPeeringConnection) || ($.eventName = RejectVpcPeeringConnection) || ($.eventName = AttachClassicLinkVpc) || ($.eventName = DetachClassicLinkVpc) || ($.eventName = DisableVpcClassicLink) || ($.eventName = EnableVpcClassicLink) }",
			shouldBeEquivalent: true,
		},

		"Same Expressions [6]": {
			expA:               "{ ($.eventSource = organizations.amazonaws.com) && (($.eventName = \"AcceptHandshake\") || ($.eventName = \"AttachPolicy\") || ($.eventName = \"CreateAccount\") || ($.eventName = \"CreateOrganizationalUnit\") || ($.eventName = \"CreatePolicy\") || ($.eventName = \"DeclineHandshake\") || ($.eventName = \"DeleteOrganization\") || ($.eventName = \"DeleteOrganizationalUnit\") || ($.eventName = \"DeletePolicy\") || ($.eventName = \"DetachPolicy\") || ($.eventName = \"DisablePolicyType\") || ($.eventName = \"EnablePolicyType\") || ($.eventName = \"InviteAccountToOrganization\") || ($.eventName = \"LeaveOrganization\") || ($.eventName = \"MoveAccount\") || ($.eventName = \"RemoveAccountFromOrganization\") || ($.eventName = \"UpdatePolicy\") || ($.eventName = \"UpdateOrganizationalUnit\")) }",
			expB:               "{ ($.eventSource = organizations.amazonaws.com) && (($.eventName = \"AcceptHandshake\") || ($.eventName = \"AttachPolicy\") || ($.eventName = \"CreateAccount\") || ($.eventName = \"CreateOrganizationalUnit\") || ($.eventName = \"CreatePolicy\") || ($.eventName = \"DeclineHandshake\") || ($.eventName = \"DeleteOrganization\") || ($.eventName = \"DeleteOrganizationalUnit\") || ($.eventName = \"DeletePolicy\") || ($.eventName = \"DetachPolicy\") || ($.eventName = \"DisablePolicyType\") || ($.eventName = \"EnablePolicyType\") || ($.eventName = \"InviteAccountToOrganization\") || ($.eventName = \"LeaveOrganization\") || ($.eventName = \"MoveAccount\") || ($.eventName = \"RemoveAccountFromOrganization\") || ($.eventName = \"UpdatePolicy\") || ($.eventName = \"UpdateOrganizationalUnit\")) }",
			shouldBeEquivalent: true,
		},

		"Same Expressions [7]": {
			expA:               "{ ($.eventName = \"ConsoleLogin\") && ($.additionalEventData.MFAUsed != \"Yes\") }",
			expB:               "{ ($.eventName = \"ConsoleLogin\") && ($.additionalEventData.MFAUsed != \"Yes\") }",
			shouldBeEquivalent: true,
		},

		"Same Expressions [8]": {
			expA:               "{ ($.eventName = \"ConsoleLogin\") && ($.additionalEventData.MFAUsed != \"Yes\") && ($.userIdentity.type = \"IAMUser\") && ($.responseElements.ConsoleLogin = \"Success\") }",
			expB:               "{ ($.eventName = \"ConsoleLogin\") && ($.additionalEventData.MFAUsed != \"Yes\") && ($.userIdentity.type = \"IAMUser\") && ($.responseElements.ConsoleLogin = \"Success\") }",
			shouldBeEquivalent: true,
		},

		"Same Expressions [9]": {
			expA:               "			{ $.userIdentity.type = \"Root\" && $.userIdentity.invokedBy NOT EXISTS && $.eventType != \"AwsServiceEvent\" }",
			expB:               "			{ $.userIdentity.type = \"Root\" && $.userIdentity.invokedBy NOT EXISTS && $.eventType != \"AwsServiceEvent\" }",
			shouldBeEquivalent: true,
		},

		"Same Expressions [10]": {
			expA:               "			{ ($.eventName=DeleteGroupPolicy)||($.eventName=DeleteRolePolicy)||($.eventName=DeleteUserPolicy)||($.eventName=PutGroupPolicy)||($.eventName=PutRolePolicy)||($.eventName=PutUserPolicy)||($.eventName=CreatePolicy)||($.eventName=DeletePolicy)||($.eventName=CreatePolicyVersion)||($.eventName=DeletePolicyVersion)||($.eventName=AttachRolePolicy)||($.eventName=DetachRolePolicy)||($.eventName=AttachUserPolicy)||($.eventName=DetachUserPolicy)||($.eventName=AttachGroupPolicy)||($.eventName=DetachGroupPolicy) }",
			expB:               "			{ ($.eventName=DeleteGroupPolicy)||($.eventName=DeleteRolePolicy)||($.eventName=DeleteUserPolicy)||($.eventName=PutGroupPolicy)||($.eventName=PutRolePolicy)||($.eventName=PutUserPolicy)||($.eventName=CreatePolicy)||($.eventName=DeletePolicy)||($.eventName=CreatePolicyVersion)||($.eventName=DeletePolicyVersion)||($.eventName=AttachRolePolicy)||($.eventName=DetachRolePolicy)||($.eventName=AttachUserPolicy)||($.eventName=DetachUserPolicy)||($.eventName=AttachGroupPolicy)||($.eventName=DetachGroupPolicy) }",
			shouldBeEquivalent: true,
		},

		"Same Expressions [11]": {
			expA:               "			{ ($.eventName = CreateTrail) || ($.eventName = UpdateTrail) || ($.eventName = DeleteTrail) || ($.eventName = StartLogging) || ($.eventName = StopLogging) }",
			expB:               "			{ ($.eventName = CreateTrail) || ($.eventName = UpdateTrail) || ($.eventName = DeleteTrail) || ($.eventName = StartLogging) || ($.eventName = StopLogging) }",
			shouldBeEquivalent: true,
		},

		"Same Expressions [12]": {
			expA:               "			{ ($.eventName = ConsoleLogin) && ($.errorMessage = \"Failed authentication\") }",
			expB:               "			{ ($.eventName = ConsoleLogin) && ($.errorMessage = \"Failed authentication\") }",
			shouldBeEquivalent: true,
		},

		"Same Expressions [13]": {
			expA:               "			{ ($.eventSource = kms.amazonaws.com) && (($.eventName=DisableKey)||($.eventName=ScheduleKeyDeletion)) }",
			expB:               "			{ ($.eventSource = kms.amazonaws.com) && (($.eventName=DisableKey)||($.eventName=ScheduleKeyDeletion)) }",
			shouldBeEquivalent: true,
		},

		"Same Expressions [14]": {
			expA:               "			{ ($.eventSource = s3.amazonaws.com) && (($.eventName = PutBucketAcl) || ($.eventName = PutBucketPolicy) || ($.eventName = PutBucketCors) || ($.eventName = PutBucketLifecycle) || ($.eventName = PutBucketReplication) || ($.eventName = DeleteBucketPolicy) || ($.eventName = DeleteBucketCors) || ($.eventName = DeleteBucketLifecycle) || ($.eventName = DeleteBucketReplication)) }",
			expB:               "			{ ($.eventSource = s3.amazonaws.com) && (($.eventName = PutBucketAcl) || ($.eventName = PutBucketPolicy) || ($.eventName = PutBucketCors) || ($.eventName = PutBucketLifecycle) || ($.eventName = PutBucketReplication) || ($.eventName = DeleteBucketPolicy) || ($.eventName = DeleteBucketCors) || ($.eventName = DeleteBucketLifecycle) || ($.eventName = DeleteBucketReplication)) }",
			shouldBeEquivalent: true,
		},

		"Same Expressions [15]": {
			expA:               "			{ ($.eventSource = config.amazonaws.com) && (($.eventName=StopConfigurationRecorder)||($.eventName=DeleteDeliveryChannel) ||($.eventName=PutDeliveryChannel)||($.eventName=PutConfigurationRecorder)) }",
			expB:               "			{ ($.eventSource = config.amazonaws.com) && (($.eventName=StopConfigurationRecorder)||($.eventName=DeleteDeliveryChannel) ||($.eventName=PutDeliveryChannel)||($.eventName=PutConfigurationRecorder)) }",
			shouldBeEquivalent: true,
		},

		"Different Spaces": {
			expA:               "{ ($.errorCode = \"*UnauthorizedOperation\") || ($.errorCode = \"AccessDenied*\") || ($.sourceIPAddress!=\"delivery.logs.amazonaws.com\") || ($.eventName!=\"HeadBucket\") }",
			expB:               "{ ($.errorCode=\"*UnauthorizedOperation\")||($.errorCode=\"AccessDenied*\")||($.sourceIPAddress!=\"delivery.logs.amazonaws.com\")||($.eventName!=\"HeadBucket\") }",
			shouldBeEquivalent: true,
		},

		"Different order of expressions": {
			expA:               "{ ($.errorCode = \"*UnauthorizedOperation\") || ($.errorCode = \"AccessDenied*\") || ($.sourceIPAddress!=\"delivery.logs.amazonaws.com\") || ($.eventName!=\"HeadBucket\") }",
			expB:               "{ ($.errorCode=\"AccessDenied*\")||($.eventName!=\"HeadBucket\")||($.errorCode=\"*UnauthorizedOperation\")||($.sourceIPAddress!=\"delivery.logs.amazonaws.com\") }",
			shouldBeEquivalent: true,
		},

		"Different logical operator": {
			expA:               "{ ($.errorCode = \"*UnauthorizedOperation\") || ($.errorCode = \"AccessDenied*\") || ($.sourceIPAddress!=\"delivery.logs.amazonaws.com\") || ($.eventName!=\"HeadBucket\") }",
			expB:               "{ ($.errorCode=\"AccessDenied*\")&&($.eventName!=\"HeadBucket\")||($.errorCode=\"*UnauthorizedOperation\")||($.sourceIPAddress!=\"delivery.logs.amazonaws.com\") }",
			shouldBeEquivalent: false,
			err:                errors.New("not supported comparison with alternating logical operators"),
		},

		"Different comparison operator": {
			expA:               "{ ($.errorCode = \"*UnauthorizedOperation\") || ($.errorCode = \"AccessDenied*\") || ($.sourceIPAddress!=\"delivery.logs.amazonaws.com\") || ($.eventName!=\"HeadBucket\") }",
			expB:               "{ ($.errorCode!=\"AccessDenied*\")||($.eventName!=\"HeadBucket\")||($.errorCode=\"*UnauthorizedOperation\")||($.sourceIPAddress!=\"delivery.logs.amazonaws.com\") }",
			shouldBeEquivalent: false,
		},

		"Different nested order": {
			expA:               "{ ($.eventSource = organizations.amazonaws.com) && (($.eventName = \"AcceptHandshake\") || ($.eventName = \"AttachPolicy\") || ($.eventName = \"CreateAccount\") || ($.eventName = \"CreateOrganizationalUnit\") || ($.eventName = \"CreatePolicy\") || ($.eventName = \"DeclineHandshake\") || ($.eventName = \"DeleteOrganization\") || ($.eventName = \"DeleteOrganizationalUnit\") || ($.eventName = \"DeletePolicy\") || ($.eventName = \"DetachPolicy\") || ($.eventName = \"DisablePolicyType\") || ($.eventName = \"EnablePolicyType\") || ($.eventName = \"InviteAccountToOrganization\") || ($.eventName = \"LeaveOrganization\") || ($.eventName = \"MoveAccount\") || ($.eventName = \"RemoveAccountFromOrganization\") || ($.eventName = \"UpdatePolicy\") || ($.eventName = \"UpdateOrganizationalUnit\")) }",
			expB:               "{ (($.eventName = \"AcceptHandshake\") || ($.eventName = \"AttachPolicy\") || ($.eventName = \"CreateAccount\") || ($.eventName = \"CreateOrganizationalUnit\") || ($.eventName = \"CreatePolicy\") || ($.eventName = \"DeclineHandshake\") || ($.eventName = \"DeleteOrganization\") || ($.eventName = \"DeleteOrganizationalUnit\") || ($.eventName = \"DeletePolicy\") || ($.eventName = \"DetachPolicy\") || ($.eventName = \"DisablePolicyType\") || ($.eventName = \"EnablePolicyType\") || ($.eventName = \"InviteAccountToOrganization\") || ($.eventName = \"LeaveOrganization\") || ($.eventName = \"MoveAccount\") || ($.eventName = \"RemoveAccountFromOrganization\") || ($.eventName = \"UpdatePolicy\") || ($.eventName = \"UpdateOrganizationalUnit\")) && ($.eventSource = organizations.amazonaws.com)}",
			shouldBeEquivalent: true,
		},

		"Different nested order (inside parenthesis)": {
			expA:               "{ ($.eventSource = organizations.amazonaws.com) && (($.eventName = \"AttachPolicy\") || ($.eventName = \"CreateAccount\") || ($.eventName = \"AcceptHandshake\") || ($.eventName = \"CreateOrganizationalUnit\") || ($.eventName = \"CreatePolicy\") || ($.eventName = \"DeclineHandshake\") || ($.eventName = \"DeleteOrganization\") || ($.eventName = \"DeleteOrganizationalUnit\") || ($.eventName = \"DeletePolicy\") || ($.eventName = \"DetachPolicy\") || ($.eventName = \"DisablePolicyType\") || ($.eventName = \"EnablePolicyType\") || ($.eventName = \"InviteAccountToOrganization\") || ($.eventName = \"LeaveOrganization\") || ($.eventName = \"MoveAccount\") || ($.eventName = \"RemoveAccountFromOrganization\") || ($.eventName = \"UpdatePolicy\") || ($.eventName = \"UpdateOrganizationalUnit\")) }",
			expB:               "{ (($.eventName = \"AcceptHandshake\") || ($.eventName = \"AttachPolicy\") || ($.eventName = \"CreateAccount\") || ($.eventName = \"CreateOrganizationalUnit\") || ($.eventName = \"CreatePolicy\") || ($.eventName = \"DeclineHandshake\") || ($.eventName = \"DeleteOrganization\") || ($.eventName = \"DeleteOrganizationalUnit\") || ($.eventName = \"DeletePolicy\") || ($.eventName = \"DetachPolicy\") || ($.eventName = \"DisablePolicyType\") || ($.eventName = \"EnablePolicyType\") || ($.eventName = \"InviteAccountToOrganization\") || ($.eventName = \"LeaveOrganization\") || ($.eventName = \"MoveAccount\") || ($.eventName = \"RemoveAccountFromOrganization\") || ($.eventName = \"UpdatePolicy\") || ($.eventName = \"UpdateOrganizationalUnit\")) && ($.eventSource = organizations.amazonaws.com)}",
			shouldBeEquivalent: true,
		},

		"Must not match on different values (empty space for string)": {
			expA:               "{ ($.eventSource = organizations.amazonaws.com) && (($.eventName = \"AttachPolicy\") || ($.eventName = \"CreateAccount\") || ($.eventName = \"AcceptHandshake\") || ($.eventName = \"CreateOrganizationalUnit\") || ($.eventName = \"CreatePolicy\") || ($.eventName = \"DeclineHandshake\") || ($.eventName = \"DeleteOrganization\") || ($.eventName = \"DeleteOrganizationalUnit\") || ($.eventName = \"DeletePolicy\") || ($.eventName = \"DetachPolicy\") || ($.eventName = \"DisablePolicyType\") || ($.eventName = \"EnablePolicyType\") || ($.eventName = \"InviteAccountToOrganization\") || ($.eventName = \"LeaveOrganization\") || ($.eventName = \"MoveAccount\") || ($.eventName = \"RemoveAccountFromOrganization\") || ($.eventName = \"UpdatePolicy\") || ($.eventName = \"UpdateOrganizationalUnit\")) }",
			expB:               "{ (($.eventName = \"AcceptHandshake  \") || ($.eventName = \"AttachPolicy\") || ($.eventName = \"CreateAccount\") || ($.eventName = \"CreateOrganizationalUnit\") || ($.eventName = \"CreatePolicy\") || ($.eventName = \"DeclineHandshake\") || ($.eventName = \"DeleteOrganization\") || ($.eventName = \"DeleteOrganizationalUnit\") || ($.eventName = \"DeletePolicy\") || ($.eventName = \"DetachPolicy\") || ($.eventName = \"DisablePolicyType\") || ($.eventName = \"EnablePolicyType\") || ($.eventName = \"InviteAccountToOrganization\") || ($.eventName = \"LeaveOrganization\") || ($.eventName = \"MoveAccount\") || ($.eventName = \"RemoveAccountFromOrganization\") || ($.eventName = \"UpdatePolicy\") || ($.eventName = \"UpdateOrganizationalUnit\")) && ($.eventSource = organizations.amazonaws.com)}",
			shouldBeEquivalent: false,
		},

		"Must match on different spacing": {
			expA:               "{ ($.eventSource = organizations.amazonaws.com           ) && (($.eventName = \"AttachPolicy\") || ($.eventName = \"CreateAccount\") || ($.eventName = \"AcceptHandshake\") || ($.eventName = \"CreateOrganizationalUnit\") || ($.eventName = \"CreatePolicy\") || ($.eventName = \"DeclineHandshake\") || ($.eventName = \"DeleteOrganization\") || ($.eventName = \"DeleteOrganizationalUnit\") || ($.eventName = \"DeletePolicy\") || ($.eventName = \"DetachPolicy\") || ($.eventName = \"DisablePolicyType\") || ($.eventName = \"EnablePolicyType\") || ($.eventName = \"InviteAccountToOrganization\") || ($.eventName = \"LeaveOrganization\") || ($.eventName = \"MoveAccount\") || ($.eventName = \"RemoveAccountFromOrganization\") || ($.eventName = \"UpdatePolicy\") || ($.eventName = \"UpdateOrganizationalUnit\")) }",
			expB:               "{ (($.eventName = \"AcceptHandshake\") || ($.eventName = \"AttachPolicy\") || ($.eventName = \"CreateAccount\") || ($.eventName = \"CreateOrganizationalUnit\") || ($.eventName = \"CreatePolicy\") || ($.eventName = \"DeclineHandshake\") || ($.eventName = \"DeleteOrganization\") || ($.eventName = \"DeleteOrganizationalUnit\") || ($.eventName = \"DeletePolicy\") || ($.eventName = \"DetachPolicy\") || ($.eventName = \"DisablePolicyType\") || ($.eventName = \"EnablePolicyType\") || ($.eventName = \"InviteAccountToOrganization\") || ($.eventName = \"LeaveOrganization\") || ($.eventName = \"MoveAccount\") || ($.eventName = \"RemoveAccountFromOrganization\") || ($.eventName = \"UpdatePolicy\") || ($.eventName = \"UpdateOrganizationalUnit\")) && ($.eventSource = organizations.amazonaws.com)}",
			shouldBeEquivalent: true,
		},

		"Must not match on logical operators": {
			expA:               "{ ($.eventSource = organizations.amazonaws.com           ) && (($.eventName = \"AttachPolicy\") || ($.eventName = \"CreateAccount\") || ($.eventName = \"AcceptHandshake\") || ($.eventName = \"CreateOrganizationalUnit\") || ($.eventName = \"CreatePolicy\") || ($.eventName = \"DeclineHandshake\") || ($.eventName = \"DeleteOrganization\") || ($.eventName = \"DeleteOrganizationalUnit\") || ($.eventName = \"DeletePolicy\") || ($.eventName = \"DetachPolicy\") || ($.eventName = \"DisablePolicyType\") || ($.eventName = \"EnablePolicyType\") || ($.eventName = \"InviteAccountToOrganization\") || ($.eventName = \"LeaveOrganization\") || ($.eventName = \"MoveAccount\") || ($.eventName = \"RemoveAccountFromOrganization\") || ($.eventName = \"UpdatePolicy\") || ($.eventName = \"UpdateOrganizationalUnit\")) }",
			expB:               "{ (($.eventName = \"AcceptHandshake\") || ($.eventName = \"AttachPolicy\") || ($.eventName = \"CreateAccount\") || ($.eventName = \"CreateOrganizationalUnit\") || ($.eventName = \"CreatePolicy\") || ($.eventName = \"DeclineHandshake\") || ($.eventName = \"DeleteOrganization\") || ($.eventName = \"DeleteOrganizationalUnit\") || ($.eventName = \"DeletePolicy\") || ($.eventName = \"DetachPolicy\") || ($.eventName = \"DisablePolicyType\") || ($.eventName = \"EnablePolicyType\") || ($.eventName = \"InviteAccountToOrganization\") || ($.eventName = \"LeaveOrganization\") || ($.eventName = \"MoveAccount\") || ($.eventName = \"RemoveAccountFromOrganization\") || ($.eventName = \"UpdatePolicy\") || ($.eventName = \"UpdateOrganizationalUnit\")) || ($.eventSource = organizations.amazonaws.com)}",
			shouldBeEquivalent: false,
		},

		"Must not match on logical operators [2]": {
			expA:               "{ ($.eventSource = organizations.amazonaws.com           ) && (($.eventName = \"AttachPolicy\") || ($.eventName = \"CreateAccount\") || ($.eventName = \"AcceptHandshake\") || ($.eventName = \"CreateOrganizationalUnit\") || ($.eventName = \"CreatePolicy\") || ($.eventName = \"DeclineHandshake\") || ($.eventName = \"DeleteOrganization\") || ($.eventName = \"DeleteOrganizationalUnit\") || ($.eventName = \"DeletePolicy\") || ($.eventName = \"DetachPolicy\") || ($.eventName = \"DisablePolicyType\") || ($.eventName = \"EnablePolicyType\") || ($.eventName = \"InviteAccountToOrganization\") || ($.eventName = \"LeaveOrganization\") || ($.eventName = \"MoveAccount\") || ($.eventName = \"RemoveAccountFromOrganization\") || ($.eventName = \"UpdatePolicy\") || ($.eventName = \"UpdateOrganizationalUnit\")) }",
			expB:               "{ (($.eventName = \"AcceptHandshake\") && ($.eventName = \"AttachPolicy\") || ($.eventName = \"CreateAccount\") || ($.eventName = \"CreateOrganizationalUnit\") || ($.eventName = \"CreatePolicy\") || ($.eventName = \"DeclineHandshake\") || ($.eventName = \"DeleteOrganization\") || ($.eventName = \"DeleteOrganizationalUnit\") || ($.eventName = \"DeletePolicy\") || ($.eventName = \"DetachPolicy\") || ($.eventName = \"DisablePolicyType\") || ($.eventName = \"EnablePolicyType\") || ($.eventName = \"InviteAccountToOrganization\") || ($.eventName = \"LeaveOrganization\") || ($.eventName = \"MoveAccount\") || ($.eventName = \"RemoveAccountFromOrganization\") || ($.eventName = \"UpdatePolicy\") || ($.eventName = \"UpdateOrganizationalUnit\")) && ($.eventSource = organizations.amazonaws.com)}",
			shouldBeEquivalent: false,
			err:                errors.New("not supported comparison with alternating logical operators"),
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			areEquivalent, err := areCloudWatchExpressionsEquivalent(tc.expA, tc.expB)
			require.Equal(t, tc.err, err)
			require.Equal(t, areEquivalent, tc.shouldBeEquivalent)
		})
	}
}

func se(l string, c comparisonOperator, r string) simpleExpression {
	return simpleExpression{
		left:     l,
		operator: c,
		right:    r,
	}
}

func ce(c logicalOperator, expressions ...expression) complexExpression {
	return complexExpression{
		operator:    c,
		expressions: expressions,
	}
}
