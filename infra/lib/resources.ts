/**
 * Resources that cannot be created in CDK for any reason
 * may be referenced by ARN or other identifier here
 */

const Resources: any = {
    global: {
        dns: {
            fullyQualifiedDomainName: "mailboxsweep.com",
            zoneId: "Z0056029HGOB514XMM3E",
        },
        api: {
            cert: {
                arn: "arn:aws:acm:us-west-2:968574982400:certificate/4a018475-220b-4f4a-99db-0f12a7950ffb",
            }
        }
    },
    staging: {
        api: {
            dns:{
                fullyQualifiedDomainName: "api.staging.mailboxsweep.com",
            },
        }
    }
};

export default Resources;