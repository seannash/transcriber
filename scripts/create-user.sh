POOL_ID=$1
aws cognito-idp admin-create-user --user-pool-id ${POOL_ID} --username bubba --user-attributes Name=email,Value=user0@example.com Name=phone_number,Value="+15555551212"     --message-action SUPPRESS
aws cognito-idp admin-set-user-password --user-pool-id ${POOL_ID}  --username bubba --password Hithere42 --permanent