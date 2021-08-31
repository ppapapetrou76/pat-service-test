# GeoIP

## Functionality

Functionality related to IP Geolocation.

### Maxmind

Uses Maxmind database GeoLite2: https://dev.maxmind.com/geoip/geoip2/geolite2/#GeoLite2_Data

The database files are stored in the database/data directory, under the country and city subdirectories. The database content is loaded into memory.

The databases can be retrieved using the following URLs. The license key is in [LastPass](https://www.lastpass.com/), please ask backend team to obtain it.

- For city database

  ```txt
  https://download.maxmind.com/app/geoip_download?edition_id=GeoLite2-City&license_key=YOUR_LICENSE_KEY&suffix=tar.gz
  ```

- For country database

  ```txt
  https://download.maxmind.com/app/geoip_download?edition_id=GeoLite2-Country&  license_key=YOUR_LICENSE_KEY&suffix=tar.gz
  ```

## Usage

- Import the desired geoip database package
- Create and use

  ```go
  import (
      "github.com/sliide/shared-go-libs/geoip/maxmind"
  )
  
  func main(){
      db := maxmind.DefaultDB()
  
      location, err := db.IPLookup(ip)
      if err != nil {
          // ...
      }
  }
  ```
