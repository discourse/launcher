package utils

import (
	"io"
	"os"
	"os/exec"
	"time"
)

const Version = "v2.2.0"

const DefaultNamespace = "local_discourse"

// Bundled plugins that we want to warn
var BundledPlugins = []string{
	"discourse-reactions",
	"discourse-apple-auth",
	"discourse-login-with-amazon",
	"discourse-lti",
	"discourse-microsoft-auth",
	"discourse-oauth2-basic",
	"discourse-openid-connect",
	"discourse-zendesk-plugin",
	"discourse-patreon",
	"discourse-graphviz",
	"discourse-rss-polling",
	"discourse-math",
	"discourse-chat-integration",
	"discourse-data-explorer",
	"discourse-post-voting",
	"discourse-user-notes",
	"discourse-staff-notes", // old name for discourse-user-notes
	"discourse-assign",
	"discourse-subscriptions",
	"discourse-hcaptcha",
	"discourse-gamification",
	"discourse-calendar",
	"discourse-question-answer", // old name for discourse-post-voting
	"discourse-adplugin",
	"discourse-affiliate",
	"discourse-github",
	"discourse-templates",
	"discourse-topic-voting",
	"discourse-policy",
	"discourse-solved",
	"discourse-ai",
}

// Known secrets, or otherwise not public info from config so we can build public images
var KnownSecrets = []string{
	"DISCOURSE_DB_HOST",
	"DISCOURSE_DB_PORT",
	"DISCOURSE_DB_SOCKET",
	"DISCOURSE_DB_REPLICA_HOST",
	"DISCOURSE_DB_REPLICA_PORT",
	"DISCOURSE_DB_PASSWORD",
	"DISCOURSE_REDIS_HOST",
	"DISCOURSE_REDIS_REPLICA_HOST",
	"DISCOURSE_REDIS_PASSWORD",
	"DISCOURSE_SMTP_ADDRESS",
	"DISCOURSE_SMTP_USER_NAME",
	"DISCOURSE_SMTP_PASSWORD",
	"DISCOURSE_DEVELOPER_EMAILS",
	"DISCOURSE_SECRET_KEY_BASE",
	"DISCOURSE_HOSTNAME",
	"DISCOURSE_SAML_CERT",
	"DISCOURSE_SAML_TITLE",
	"DISCOURSE_SAML_TARGET_URL",
	"DISCOURSE_SAML_NAME_IDENTIFIER_FORMAT",
}

func findDockerPath() string {
	location, err := exec.LookPath("docker.io")
	if err != nil {
		location, _ := exec.LookPath("docker")
		return location
	}
	return location
}

var DockerPath = findDockerPath()

var Out io.Writer = os.Stdout

var CommitWait = 2 * time.Second
