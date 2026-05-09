#### in ./sandbox/ we have a functional proof od concept of the value proposition

#### want to build ultra-minimalistic micro-service

## this is for ultra optimization of PSN friends management

with style inspired (copied) from /Users/etozzato/WorkSpace/_flutter/plain for the UI/UX and patterns (for example the theming)
and a tailored golang backend and Dockerized stack ready to deploy to my linux laptop-server inspired by (copied) /Users/etozzato/WorkSpace/_flutter/plain_radio/server
and /Users/etozzato/WorkSpace/PlayStation/campari. Some ideas for the Dockerization might come from the very advanced setup of /Users/etozzato/WorkSpace/_AINZ/PharmWare-61
We should use postgres. This will be a real world micro app. This might need an emergency admin panel ready for editing entries. Text based like for the radio admin: list the groups created. delete, rename, ban. This app might need to be able to run with SSL. I think campari has the most up to date nginx multi-microservice deployment.
let's document this very well. detailed but essential documentation!

USE CASE

a user goes to /new
creates a new group (<= name[unique])
receives a long-sha (=> URL[unique])
if user gives email, receives* admin link. can re-request admin link
if user refuses email, receives* one secret key (?) and might lose access to group
with link you can add entry (<=PSN_ID)
when you add entry, you receive* a unique admin link for the entry (=> URL[unique])
admin of the group can admin any entry
admin can add entry to a block list for the group
backend should analyze a new entry on change and determine if profile is public
if public, entry will be displayed as a ps-app compatible permalink - such https://profile.playstation.com/#{CGI.escape(online_id)}
if private, entry will be a different url. to be determined if google search + profile or just the handle copied to the clipboard.

content for GET /app-url/SHA/
* minimal header (sticky) group name + sort (A-Z or recent) + NEW
* list of cute, well designed cards in a grid / column on the page. QR + name / profile name this needs to look good!
* cards need to be very spaced, perhaps vertically so there is no risk of over-scanning

content for GET /app-url/SHA/psn-id
* minimal header (sticky) group name + NEW | PULL | REMOVE | BAN (for admin)
* single card

public/private profile are cached as postgres bool and we will need an endpoint to trigger re-fetch and eval
for example /app-url/SHA/psn-id/pull
POST to /app-url/SHA/psn-id/remove and /ban need to deal with some sort "AM I ADMIN BECAUSE I LANDED ON THIS PAGE WITH THE SECRET TOKEN MATCHING THIS ENTRY RECORD"

*received = given on screen

## UPDATE 1

 /app-url/SHA/upload is available for admin: only a 2 column CSV NAME(optional),PSN-ID for batch add

 # UPDATE 2

 we mentioned email - do we have a way to send an email? if not, let's ditch that idea and add a "EMAIL YOURSELF" button: wherever the private key is shown, after any creation the EMAIL YOURSELF button opens their own configured email client with subject: XXX and Body: YYY (you know how to fill in the blanks)
