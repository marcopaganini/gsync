*NAME*

gsync - Sync files between local filesystems and Google Drive

*SYNOPSIS*

gsync [OPTION] source... destination

*DESCRIPTION*

Sync files and directories between the local filesystem and a Google Drive location.
This program accepts multiple sources, and it's even possible to sync a local filesystem
to a local filesystem or a Gdrive location to another Gdrive location.

Sources and destination are conventional paths like unix paths. Sources can specify
a file or a directory, in which case all files inside the directory will be copied.
If the source directory ends in "/" (slash), then all files inside that directory
will be copied to destination. Otherwise, gsync will create the source directory
inside the destination, and copy all files.

For the moment, only files and directories are supported and permissions are not kept.
This will change in future releases.

The program considers anything that looks like a local path to be local. Google Drive
paths should start with "g:" or "gdrive:". In Google drive, paths always start from
root, so the initial slash in a path is not necessary.

Options:

*--dry-run*
*-n*

Simulate the operation (dry-run)

*--verbose*
*-v*

Verbose Mode. Without this, only error and warning messages will be printed.

*--id*
*--secret*
*--code*

These options are used during initial setup, to pass the required Oauth credentials
for use with Google Drive. To set up your Google Drive account, visit the
[Google Developers Page](https://developers.google.com/drive/web/enable-sdk) to
create the Id and Secret. Run gsync with the --id _yourid_ and --secret _yoursecret_
Gsync will prompt for a code and provide an URL. Visit that URL and repeat the
gsync command adding --code _yourcode_. Credentials will be saved locally and future
invocations of gsync won't require these flags.

*NOTES*

Things are changing fast and features are being added daily.

*AUTHOR*

(C) Aug/2014 by Marco Paganini <paganini AT paganini DOT net>
