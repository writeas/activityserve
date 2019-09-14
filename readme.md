# ActivityServe

## A very light ActivityPub library in go

This library was built to support the very little functions that [pherephone](https://github.com/writeas/pherephone) requires. It might never be feature-complete but it's a very good point to start your activityPub journey. Take a look at [activityserve-example] for a simple main file that uses **activityserve** to post a "Hello, world" message.

For now it supports following and unfollowing users, accepting follows, announcing (boosting) other posts and this is pretty much it. 

The library is still a moving target and the api is not guaranteed to be stable.

You can override the auto-accept upon follow by setting the `actor.OnFollow` to a custom function. 