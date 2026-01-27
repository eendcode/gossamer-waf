# Gossamer

Gossamer is a web application firewall, aimed to protect web applications and hinder, confuse and disrupt attackers via unconventional methods.

## Under the hood: Coraza

We use [Coraza](https://github.com/corazawaf/coraza) with the [core rule set](https://github.com/coreruleset/coreruleset) for stopping attacks, together with some rulesets of our own.

## Hinder the attackers

A WAF does not stop all attacks all the time. We can put in some more effort to stop attackers in their tracks. For example,

- We can enforce **tracking cookies** onto visitors. Without it, the WAF stops you. Getting a cookie may require "proving" that you're a regular user, or are doing a very good job at imposing as one.
- We may **rate limit** or **throttle** traffic from visitors that have set off the WAF previously in their session.

## Logging

Gossamer logs to JSON by default, to stdout.
