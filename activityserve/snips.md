## When we follow someone from pherephone 1.00

``` json

{
    "@context": "https://www.w3.org/ns/activitystreams",
    "actor": "https://floorb.qwazix.com/myAwesomeList1",
    "id": "https://floorb.qwazix.com/myAwesomeList1/Xm9UHyJXyFYduqXz",
    "object": "https://cybre.space/users/qwazix",
    "to": "https://cybre.space/users/qwazix",
    "type": "Follow"
}
```

``` yaml

Accept: application/activity+json 
Accept-Charset: utf-8
Date: Tue, 10 Sep 2019 05:31:22 GMT 
Digest: SHA-256=uL1LvGU4+gSDm8Qci6XibZODTaNCsXWXWgkMWAqBvG8= 
Host: cybre.space
Signature: keyId="https://floorb.qwazix.com/myAwesomeList1#main-key",algorithm="rsa-sha256",headers="(request-target) date host digest",signature="c6oipeXu/2zqX3qZF1x7KLNTYifcyqwwDySoslAowjpYlKWO3qAZMU1A//trYm23AtnItXkH2mY3tPq8X7fy9P1+CMFmiTzV01MGwwwJLDtEXKoq8W7L7lWuQhDD5rjiZqWyei4T13FW7MOCRbAtC4kZqkHrp5Z3l8HhPvmgUV5VOuSGWrtbmCN3hlAEHVugQTMPC6UjlaHva6Qm/SNlFmpUdG7WmUUPJIZ6a/ysBk4cLkF1+Hb03grXKexLHAU4bPIRcjwFpUl06yp8fZ8CCLhNhIsBACiizV85D3votmdxAollE5JXSwBp4f6jrZbgiJEusFoxiVKKqZRHRESQBQ=="

```

## Pherephone 1 Accept Activity

``` yaml
    Accept: application/activity+json
    Accept-Charset: utf-8
    Date: Tue, 10 Sep 2019 07:28:49 GMT
    Digest: SHA-256=GTy9bhYjOnbeCJzAzpqI/HEw/5p81NnoPLJkVAiZ4K0=
    Host: cybre.space
    Signature: keyId="https://floorb.qwazix.com/activityserve_test_actor_1#main-key",algorithm="rsa-sha256",headers="(request-target) date host digest",signature="jAeTEy9v1t+bCwQJB2R4Cscu/fGu5i4luHXlzJcJVyRbsHGqxbNEOxlk/G0S5BGbX3Kuoerq2oMpkFV5kCWPlpAmfhz38NKIrWhjnEUpFOfiG+ZJBpQsb3VQp7M3RGPZ9K4hmV6BSzkC8npsFGPI/HkAaj9u/txW5Cp4v6dMOYteoRLcKc3UVPK9j4hCbjq6SPhpwfM+StARSDnUFfpDe4YYQiVnO2WoINPUr4xvELmCYdBclSBCKcG66g8sBpnx4McjIlu0VISeBxzIHZYOONPteLY2uZW3Axi9JIAq88Y2Ecw4vV6Ctp7KcmD7M3kAJLqao2p/XZNZ3ExsTGfrXA=="
    User-Agent: activityserve 0.0
```

``` json
{
    "@context": "https://www.w3.org/ns/activitystreams",
    "actor": "https://floorb.qwazix.com/myAwesomeList1",
    "id": "https://floorb.qwazix.com/myAwesomeList1/SABRE7xlDAjtDcZb",
    "object": {
        "actor": "https://cybre.space/users/qwazix",
        "id": "https://cybre.space/3e7336af-4bcd-4f77-aa69-6a145be824aa",
        "object": "https://floorb.qwazix.com/myAwesomeList1",
        "type": "Follow"
    },
    "to": "https://cybre.space/users/qwazix",
    "type": "Accept"
}
```

## Pherephone 2 Accept Activity

``` yaml

Accept: application/activity+json
Accept-Charset: utf-8
Date: Tue, 10 Sep 2019 07:32:08 GMT
Digest: SHA-256=yKzA6srSMx0b5GXn9DyflXVdqWd6ADBGt5hO9t/yc44=
Host: cybre.space
Signature: keyId="https://floorb.qwazix.com/myAwesomeList1#main-key",algorithm="rsa-sha256",headers="(request-target) date host digest",signature="WERXWDRFS7aGiIoz+HSujtuv9XNFBPxHkJSsCPu7PNIUDoAB2jdwW3rZc5jbrSLxi9Aqhr2BiBV/VYELQ8gITPzzIYH5sizPcPyLyARPUw37t6zA3HinahpfBKXhf73q9u+CYE/7DMKQ2Pvv2lQPaZ8hl27R2KJmcc3Jhmn5nxrQ+kxAtn6qYpNT/BqLWlXKx5rpYM2r+mHjFyYRYsjlAmi+RQNDEmv/uwn+XuNKzEtrL8Oq7mM13Lsid0a3gJi/t0b/luoyRyvi3fHUM/b1epfVogG/FulsZ0A92310v8MbastceQjjUzTzjKHILl7qNewkqtlzn2ARm3cZlAprSg=="
User-Agent: pherephone (go-fed/activity v1.0.0)


```

``` json

{
        "@context": "https://www.w3.org/ns/activitystreams",
        "actor": "https://floorb.qwazix.com/activityserve_test_actor_1",
        "id": "https://floorb.qwazix.com/activityserve_test_actor_1/4wJ9DrBab4eIE3Bt",
        "object": {
                "actor": "https://cybre.space/users/qwazix",
                "id": "https://cybre.space/9123da78-21a5-44bc-bce5-4039a4072e4c",
                "object": "https://floorb.qwazix.com/activityserve_test_actor_1",
                "type": "Follow"
        },
        "to": "https://cybre.space/users/qwazix",
        "type": "Accept"
} 

```

