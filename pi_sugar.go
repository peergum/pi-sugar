/*
   pi_sugar,
   Copyright (C) 2024  Phil Hilger

   This program is free software: you can redistribute it and/or modify
   it under the terms of the GNU General Public License as published by
   the Free Software Foundation, either version 3 of the License, or
   (at your option) any later version.

   This program is distributed in the hope that it will be useful,
   but WITHOUT ANY WARRANTY; without even the implied warranty of
   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
   GNU General Public License for more details.

   You should have received a copy of the GNU General Public License
   along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

package pi_sugar

import (
	"github.com/peergum/go-rpio/v5"
	"log"
)

type PiSugar struct {
	voltage     float64
	charge      int
	power       bool
	charging    bool
	model       int
	temperature int
	*rpio.I2cDevice
}

const (
	piSugarI2CAddress = 0x57

	powerReg         = 0x02
	temperatureReg   = 0x04
	voltageReg       = 0x22
	batteryChargeReg = 0x2a
	//chargingStatusReg
	secondsInAMinute = 60
	minutesInAnHour  = 60
	hoursInADay      = 24
	numberOfDays     = 7
)

var (
	piSugar               PiSugar
	lastMinuteCharge      []int     = make([]int, 0, secondsInAMinute)
	lastHourCharge        []float64 = make([]float64, 0, minutesInAnHour)
	lastDayCharge         []float64 = make([]float64, 0, hoursInADay*numberOfDays)
	lastMinuteVoltage     []float64 = make([]float64, 0, secondsInAMinute)
	lastHourVoltage       []float64 = make([]float64, 0, minutesInAnHour)
	lastDayVoltage        []float64 = make([]float64, 0, hoursInADay*numberOfDays)
	lastMinuteTemperature []int     = make([]int, 0, secondsInAMinute)
	lastHourTemperature   []float64 = make([]float64, 0, minutesInAnHour)
	lastDayTemperature    []float64 = make([]float64, 0, hoursInADay*numberOfDays)
	counter               int
)

func Init() (err error) {
	if err = rpio.Open(); err != nil {
		log.Printf("Can't open rpio %v", err)
		return err
	}

	if piSugar.I2cDevice, err = rpio.I2cBegin(rpio.I2c1, piSugarI2CAddress); err != nil {
		log.Printf("Can't start I2C %v", err)
		return err
	}
	piSugar.I2cSetSlaveAddress(0x57)
	//piSugar.I2cSetBaudrate(110000)
	return nil
}

func End() {
	piSugar.I2cEnd()
}

func NewPiSugar() (*PiSugar, error) {
	return &piSugar, nil
}

func (piSugar *PiSugar) Voltage() float64 {
	return piSugar.voltage
}

func (piSugar *PiSugar) Charge() int {
	return piSugar.charge
}

func (piSugar *PiSugar) Charging() bool {
	return piSugar.charging
}

func (piSugar *PiSugar) Power() bool {
	return piSugar.power
}

func appendInt(table []int, value int, maxSize int) []int {
	firstElement := 0
	if len(table) == maxSize {
		firstElement = 1
	}
	table = append(table[firstElement:], value)
	return table
}

func appendFloat64(table []float64, value float64, maxSize int) []float64 {
	firstElement := 0
	if len(table) == maxSize {
		firstElement = 1
	}
	table = append(table[firstElement:], value)
	return table
}

func avgInt(table []int) (avg float64) {
	for _, v := range table {
		avg += float64(v)
	}
	return avg / float64(len(table))
}

func avgFloat64(table []float64) (avg float64) {
	for _, v := range table {
		avg += v
	}
	return avg / float64(len(table))
}

func (piSugar *PiSugar) Refresh() {
	var buf []byte = make([]byte, 2)
	counter++

	// we keep history of each variable
	// 60 last seconds
	// 60 last minutes
	// "numberOfDays" last days
	code := piSugar.I2cReadRegister(temperatureReg, buf, 1)
	if code == 0 {
		lastMinuteTemperature = appendInt(lastMinuteTemperature, int(buf[0])-40, secondsInAMinute)
		piSugar.temperature = int(avgInt(lastMinuteTemperature))
		if counter%60 == 0 {
			lastHourTemperature = appendFloat64(lastHourTemperature, avgInt(lastMinuteTemperature), minutesInAnHour)
			if counter%1440 == 0 {
				lastDayTemperature = appendFloat64(lastDayTemperature, avgFloat64(lastHourTemperature), numberOfDays*hoursInADay)
			}
		}
	}
	code = piSugar.I2cReadRegister(voltageReg, buf, 2)
	if code == 0 {
		lastMinuteVoltage = appendFloat64(lastMinuteVoltage, float64(uint16(buf[0])<<8|uint16(buf[1]))/1000, secondsInAMinute)
		piSugar.voltage = avgFloat64(lastMinuteVoltage)
		if counter%60 == 0 {
			lastHourVoltage = appendFloat64(lastHourVoltage, avgFloat64(lastMinuteVoltage), minutesInAnHour)
			if counter%1440 == 0 {
				lastDayVoltage = appendFloat64(lastDayVoltage, avgFloat64(lastHourVoltage), numberOfDays*hoursInADay)
			}
		}
	}
	code = piSugar.I2cReadRegister(batteryChargeReg, buf, 1)
	if code == 0 {
		lastMinuteCharge = appendInt(lastMinuteCharge, int(buf[0]), secondsInAMinute)
		piSugar.charge = int(avgInt(lastMinuteCharge))
		if counter%60 == 0 {
			lastHourCharge = appendFloat64(lastHourCharge, avgInt(lastMinuteCharge), minutesInAnHour)
			if counter%1440 == 0 {
				lastDayCharge = appendFloat64(lastDayCharge, avgFloat64(lastHourCharge), numberOfDays*hoursInADay)
			}
		}
	}
	code = piSugar.I2cReadRegister(powerReg, buf, 1)
	if code == 0 {
		piSugar.power = buf[0]&0x80 != 0
	}
	Debug("T = %dºC, V = %.3fV, B = %d%%, P = %t",
		piSugar.temperature,
		piSugar.voltage,
		piSugar.charge,
		piSugar.power)
}
