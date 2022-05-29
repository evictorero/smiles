# Smiles 
Consultar la API de Smiles buscando los vuelos más baratos con millas

### Como usarlo
- Descargar del último release el programa compatible con tu sistema operativo y arquitectura
- Ejecutar por línea de comando enviando parámetros: `Origen Destino FechaDeSalida FechaDeVuelta DíasCorridosAConsultar`
- - Ejemplo: `smiles EZE PUJ 2022-09-10 2022-09-20 5`

### Ejecutar directamente desde el código fuente
- Instalar Go
- Clonar el repositorio
- Configurar parámetros al comienzo de `main.go`
- Setear flag `useCommandLineArguments` en `false` 
- `go run main.go`

