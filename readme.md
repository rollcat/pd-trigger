# `pd-trigger` - trigger PagerDuty incidents from the command line

This is a simple CLI client for [PagerDuty](https://www.pagerduty.com),
that has one and only one purpose: trigger incidents/alerts. It does
not do absolutely anything else, and in this simplicity hopes to
provide maximum value.

## Usage

To trigger an incident:

    pd-trigger "Something went horribly wrong."

To set a higher severity (default is `info`):

    pd-trigger -si "This is for your information."
    pd-trigger -se "This is an error."
    pd-trigger -sw "This is a warning."
    pd-trigger -sc "This is a crticial condition."

To avoid creating duplicate incidents, set a deduplication key:

    pd-trigger -k some-key "This event is deduplicated."

To override the source of the event (by default, hostname is used):

    pd-trigger -s coffee-maker "The brew failed."

## Installation

```shell
go install github.com/rollcat/pd-trigger@latest
```

The executable name is `pd-trigger`.

## Setup

The setup instructions are also available in the program itself:
`pd-trigger --help-setup`.

1. Generate the auth token:
    - Open your Pagerduty dashboard
    - Integrations -> API Access Keys -> Create New API Key
    - Description: pd-trigger
    - Create Key

2. Generate the integration key:
    - Open your Pagerduty dashboard
    - Services -> Service Directory
    - Select a service, or create a new one
    - Integrations
    - "Events API V2" (create the integration if it does not exist)
    - Integration Key

3. Create `~/.config/pagerduty.yml` (per `XDG_CONFIG_HOME`),
   or `/etc/xdg/pagerduty.yml` (per `XDG_CONFIG_DIRS`),
   with the following contents:

    ```yaml
    authtoken: "<your auth token>"
    integrationkey: "<your integration key>"
    ```

If the config file `~/.pd.yml` exists, it will also be tried; it
provides limited compatibility with the [official Go commandline
client](https://github.com/PagerDuty/go-pagerduty).

## Author

&copy; 2023 Kamil Cholewiński <<kamil@rollc.at>>

License is [MIT](/license.txt).
