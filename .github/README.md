## ![img](https://sakura.rex.wf/linear/squiggle?size=36) squiggle

squiggle is a tiny self hostable llm-moderated guestbook backend built with golang and echo. comes with a real time admin panel.

### local quickstart 

1. fill `.env` with keys (look at `.env.sample`)

2. run with docker

```
docker compose -f docker-compose.dev.yml up --build
```

3. (optional) generate and seed random data

```
go run ./sql/seed-gen.go
docker compose exec -T postgres psql -U <DB_USER> -d <DB_NAME> < ./seed.sql
```

4. to build for prod

```
docker compose up --build --watch
```

### to use in your static site

1. attach turnstile

```html
  <script src="https://challenges.cloudflare.com/turnstile/v0/api.js" async defer></script>

  <div class="cf-turnstile" data-sitekey="1x00000000000000000000AA"></div>
```

2. make a post request at `/entry`

```js
const turnstileToken = document.querySelector('[name="cf-turnstile-response"]').value;
const res = await fetch(BASE_URL + "/entry", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
        name,
        email,
        site,
        message,
        turnstileToken,
    }),
});
```


an example is given in `/mock/index.html`
