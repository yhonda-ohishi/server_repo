// Minimal Swagger UI Bundle JavaScript placeholder
// In production, this should be replaced with the actual Swagger UI bundle

console.log('Loading Swagger UI...');

// Minimal SwaggerUIBundle implementation for demonstration
window.SwaggerUIBundle = function(config) {
    console.log('SwaggerUIBundle initialized with config:', config);

    // Create a basic UI structure
    const container = document.querySelector(config.dom_id);
    if (container) {
        container.innerHTML = `
            <div class="loading">
                <h1>ETC Meisai API Documentation</h1>
                <p>Loading API specification from: ${config.url}</p>
                <p><strong>Note:</strong> This is a minimal implementation.
                In production, replace with the full Swagger UI bundle from
                <a href="https://github.com/swagger-api/swagger-ui">swagger-ui</a></p>
                <p><a href="${config.url}">View OpenAPI Specification (JSON)</a></p>
            </div>
        `;

        // Fetch and display the API spec
        fetch(config.url)
            .then(response => response.json())
            .then(spec => {
                const specHtml = '<pre>' + JSON.stringify(spec, null, 2) + '</pre>';
                container.innerHTML = `
                    <div class="info">
                        <h1>${spec.info.title}</h1>
                        <p>${spec.info.description}</p>
                        <p>Version: ${spec.info.version}</p>
                    </div>
                    <div>
                        <h2>API Specification</h2>
                        ${specHtml}
                    </div>
                `;

                if (config.onComplete) {
                    config.onComplete();
                }
            })
            .catch(error => {
                console.error('Failed to load API spec:', error);
                if (config.onFailure) {
                    config.onFailure(error);
                }
            });
    }

    return {};
};

// Minimal preset
window.SwaggerUIStandalonePreset = {};

// Note: In production, download and use the actual Swagger UI files:
// - swagger-ui-bundle.js
// - swagger-ui-standalone-preset.js
// - swagger-ui.css
// - swagger-ui-bundle.css
// From: https://github.com/swagger-api/swagger-ui/tree/master/dist