# TODO

## Security best practices

### Cross-Site Request Forgery (CSRF) Protection
If you add any forms or allow users to change state on the server, you must protect against CSRF attacks. This attack tricks a user's browser into making a request to your server that they didn't intend to. Libraries like gorilla/csrf (https://github.com/gorilla/csrf) can help you implement this protection.
