# S4: Stupidly Simple Storage Service

Why? Somehow trying to get rsync to behave when yeeted into a docker container felt like trying to push a small cylinder into a square hole without any peanut butter.

Also because it was definitely the least soul sucking option out of them two, when the other one was to use a self-hosted S3 server together with the AWS SDK.

It will likely cause more "OH FUCK" moments than it really should, but at least I understand why it’s broken.

And no, the name is not an S-Bahn reference. Even though allegedly I like trains a bit more than a normal person should.

## Architecture (If you can call it that)

The system is split into two parts. They talk to each other over a REST API I’ve titled **s4**. It’s not revolutionary; it’s just the bare minimum required to move a file from Point A to Point B without losing my mind.

1. **The Server:** The lonely box that holds your files.
2. **The Client:** The thing that yells at the server.

---

## Deployment (Server)

Getting the server running is "easy," provided you have a basic grasp of how computers work.

* **Docker:** Just spin up the container. If you don't know how to do that, there are plenty of tutorials online that I didn't write.
* **Configuration:** There’s a config file. I’ll provide it separately. You’ll need to "configure some stuff" in it. I trust you can handle that without a 50-page manual, but maybe I’m being optimistic.

## Usage (Client)

Once the server is running—assuming you configured it correctly—the client can move files.

### 1. Tell the Client where to go

The client isn't psychic. You actually have to tell it the **Server URL (Remote)** and the **User Credentials** you set up in the config file. If you forget these, the client will just sit there, and honestly, I don't want to hear about it.

### 2. The Commands

There are two commands. I kept them simple so there’s less for you to mess up:

* `push`: Sends a file to the server.
* `pull`: Gets a file from the server.

### 3. Storage Logic

Depending on how you (mis)configured the server, clients might have:

* **Shared Access:** Everyone sees everything. Total chaos.
* **Separate Roots:** Everyone stays in their own little sandbox.
* **Mix of both:** I mean, nobody is stopping you from making this more complex than my past relationship

This depends entirely on your server settings. If everyone is overwriting each other's files, please refer back to the **"I’m Not Your Mother" Clause** in the license.

---

## Troubleshooting

If it doesn't work:

1. Double-check your config file.
2. Check the logs. There is a good chance it tells you exactly what the fuck is wrong.
4. **Check your OS:** Look, I wrote this on Linux, for Linux. If you are trying to run this on Windows and you're seeing weird backslash errors, permission loops, or the blue screen of "Why Did I Buy This?", that’s between you and Microsoft. I’m not saying it *won’t* work on Windows, I’m just saying I didn't care enough to check.
