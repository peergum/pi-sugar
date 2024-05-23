//module github.com/peergum/go-pisugar
module pi_sugar

go 1.22

require github.com/peergum/go-rpio/v4 v4.0.0-20240502133125-d1c231628b14

replace github.com/peergum/go-rpio/v4 v4.0.0-20240502133125-d1c231628b14 => ./go-rpio
