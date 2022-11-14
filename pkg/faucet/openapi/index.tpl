<!--HTML Snippet used from https://github.com/swagger-api/swagger-ui/blob/1bb70a299650773e7d7416b8b3e0b251bf6d8c93/docs/usage/installation.md#unpkg-->
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <meta
    name="description"
    content="SwaggerUI"
  />
  <title>SwaggerUI</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@4.5.0/swagger-ui.css" />
</head>
<body>
<div id="swagger-ui"></div>
<script src="https://unpkg.com/swagger-ui-dist@4.5.0/swagger-ui-bundle.js" crossorigin></script>
<script>
  window.onload = () => {
    window.ui = SwaggerUIBundle({
                url: {{ .URL }},
                dom_id: "#swagger-ui",
                deepLinking: true,
                layout: "BaseLayout",
    });
  };
</script>
</body>
</html>